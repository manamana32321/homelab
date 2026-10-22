#!/usr/bin/env python3
import argparse
import copy
import glob
import os
import subprocess
import sys
import tempfile
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass, field

import yaml

REPO_URL = "https://github.com/manamana32321/homelab"
APPS_GLOB = "k8s/argocd/applications/**/*.yaml"
API_VERSIONS_FILE = ".github/helm-api-versions.txt"
SUC_PLANS = "k8s/system-upgrade/plans.yaml"
TRAEFIK_CONFIG = "k8s/traefik/manifests/helmchartconfig.yaml"
K3S_CHARTS_REPO = "https://k3s-io.github.io/k3s-charts"
TOOL_FILES = {"scripts/helm-dead-values.py", API_VERSIONS_FILE, SUC_PLANS, ".github/workflows/k8s-validate.yaml"}
MISSING = object()
FAILING = {"DEAD", "CONDITIONAL"}


@dataclass
class Target:
    name: str
    source_file: str
    repo: str
    chart: str
    version: str
    namespace: str
    release: str
    values_files: list = field(default_factory=list)
    inline_values: dict = field(default_factory=dict)
    base_values: dict = field(default_factory=dict)


def load_yaml(path):
    with open(path) as f:
        return yaml.safe_load(f) or {}


def load_docs(path):
    with open(path) as f:
        return [d for d in yaml.safe_load_all(f) if d]


def deep_merge(a, b):
    out = dict(a)
    for k, v in b.items():
        out[k] = deep_merge(out[k], v) if isinstance(out.get(k), dict) and isinstance(v, dict) else v
    return out


def argocd_targets():
    targets = []
    for path in sorted(glob.glob(APPS_GLOB, recursive=True)):
        for doc in load_docs(path):
            if doc.get("kind") != "Application":
                continue
            spec = doc["spec"]
            sources = spec.get("sources") or [spec["source"]]
            refs = {s["ref"]: s["repoURL"] for s in sources if s.get("ref")}
            for src in sources:
                if not src.get("chart"):
                    continue
                helm = src.get("helm") or {}
                files = []
                for vf in helm.get("valueFiles") or []:
                    ref, _, rel = vf.partition("/")
                    if ref.startswith("$") and refs.get(ref[1:], "").rstrip("/").removesuffix(".git") == REPO_URL:
                        files.append(rel)
                inline = helm.get("valuesObject") or yaml.safe_load(helm.get("values") or "") or {}
                targets.append(Target(
                    name=doc["metadata"]["name"],
                    source_file=path,
                    repo=src["repoURL"],
                    chart=src["chart"],
                    version=str(src["targetRevision"]),
                    namespace=spec["destination"].get("namespace", "default"),
                    release=helm.get("releaseName") or doc["metadata"]["name"],
                    values_files=files,
                    inline_values=inline,
                ))
    return targets


def k3s_version():
    for doc in load_docs(SUC_PLANS):
        if doc["metadata"]["name"] == "k3s-server":
            return doc["spec"]["version"]
    sys.exit(f"k3s-server plan not found in {SUC_PLANS}")


def traefik_target(k3s):
    url = f"https://raw.githubusercontent.com/k3s-io/k3s/{k3s.replace('+', '%2B')}/manifests/traefik.yaml"
    manifest = subprocess.run(["curl", "-sfL", url], capture_output=True, text=True, check=True).stdout
    chart = next(d for d in yaml.safe_load_all(manifest) if d and d["metadata"]["name"] == "traefik")
    version = chart["spec"]["chart"].rsplit("/", 1)[1].removeprefix("traefik-").removesuffix(".tgz")
    return Target(
        name="traefik (k3s HelmChartConfig)",
        source_file=TRAEFIK_CONFIG,
        repo=K3S_CHARTS_REPO,
        chart="traefik",
        version=version,
        namespace="kube-system",
        release="traefik",
        inline_values=yaml.safe_load(load_yaml(TRAEFIK_CONFIG)["spec"]["valuesContent"]) or {},
        base_values=yaml.safe_load(chart["spec"].get("valuesContent") or "") or {},
    )


def pull_chart(t, cache):
    dest = os.path.join(cache, f"{t.chart}-{t.version}")
    chart_dir = os.path.join(dest, t.chart)
    if os.path.isdir(chart_dir):
        return chart_dir
    if t.repo.startswith("http"):
        cmd = ["helm", "pull", t.chart, "--repo", t.repo]
    else:
        cmd = ["helm", "pull", f"oci://{t.repo.removeprefix('oci://').rstrip('/')}/{t.chart}"]
    subprocess.run(cmd + ["--version", t.version, "--untar", "--untardir", dest], check=True, capture_output=True)
    return chart_dir


def chart_defaults(chart_dir):
    vals = load_yaml(os.path.join(chart_dir, "values.yaml")) if os.path.isfile(os.path.join(chart_dir, "values.yaml")) else {}
    meta = load_yaml(os.path.join(chart_dir, "Chart.yaml"))
    alias = {d["name"]: d.get("alias", d["name"]) for d in meta.get("dependencies") or []}
    sub_root = os.path.join(chart_dir, "charts")
    if os.path.isdir(sub_root):
        for name in os.listdir(sub_root):
            sub = os.path.join(sub_root, name)
            if not os.path.isfile(os.path.join(sub, "Chart.yaml")):
                continue
            key = alias.get(load_yaml(os.path.join(sub, "Chart.yaml"))["name"], name)
            cur = vals.get(key)
            vals[key] = deep_merge(chart_defaults(sub), cur if isinstance(cur, dict) else {})
    return vals


class Renderer:
    def __init__(self, chart_dir, t, helm_args):
        self.cmd = ["helm", "template", t.release, chart_dir, "-n", t.namespace, "--include-crds", *helm_args]
        self.base = t.base_values
        self.volatile = set()

    def raw(self, values):
        with tempfile.TemporaryDirectory() as d:
            base_f, vals_f = os.path.join(d, "base.yaml"), os.path.join(d, "values.yaml")
            with open(base_f, "w") as f:
                yaml.safe_dump(self.base, f)
            with open(vals_f, "w") as f:
                yaml.safe_dump(values, f)
            out = subprocess.run(self.cmd + ["-f", base_f, "-f", vals_f], capture_output=True, text=True)
        if out.returncode != 0:
            return None, out.stderr.strip()
        return out.stdout, None

    def learn_volatile(self, values):
        a, err = self.raw(values)
        if err:
            return err
        b, _ = self.raw(values)
        for line in set(a.splitlines()) ^ set(b.splitlines()):
            self.volatile.add(line.split(":", 1)[0])
        return None

    def render(self, values):
        out, err = self.raw(values)
        if err:
            return "ERROR " + err
        return "\n".join(
            line.split(":", 1)[0] + ": <volatile>" if line.split(":", 1)[0] in self.volatile else line
            for line in out.splitlines()
        )


def without(values, path):
    v = copy.deepcopy(values)
    cur = v
    for k in path[:-1]:
        cur = cur[k]
    del cur[path[-1]]
    return v


def mutate(v):
    if isinstance(v, dict):
        return {k: mutate(x) for k, x in v.items()}
    if isinstance(v, list):
        return [mutate(x) for x in v] + ["helm-dead-values-probe"]
    if isinstance(v, bool):
        return not v
    if isinstance(v, (int, float)):
        return v + 7
    return f"{v}-helm-dead-values-probe"


def replaced(values, path, new):
    v = copy.deepcopy(values)
    cur = v
    for k in path[:-1]:
        cur = cur[k]
    cur[path[-1]] = new
    return v


def lookup(tree, path):
    cur = tree
    for k in path:
        if not isinstance(cur, dict) or k not in cur:
            return MISSING
        cur = cur[k]
    return cur


def matches_default(ours, default):
    if isinstance(ours, dict) and isinstance(default, dict):
        return all(k in default and matches_default(v, default[k]) for k, v in ours.items())
    return ours == default


def analyze(t, cache, helm_args):
    values = {}
    for f in t.values_files:
        values = deep_merge(values, load_yaml(f))
    values = deep_merge(values, t.inline_values)
    if not values:
        return t, [], None
    try:
        chart_dir = pull_chart(t, cache)
    except subprocess.CalledProcessError as e:
        return t, [], f"chart pull failed: {e.stderr.decode().strip()}"
    r = Renderer(chart_dir, t, helm_args)
    err = r.learn_volatile(values)
    if err:
        return t, [], f"render failed: {err}"
    baseline = r.render(values)
    defaults = deep_merge(chart_defaults(chart_dir), t.base_values)
    findings = []

    def ignored(node, path):
        if lookup(defaults, path) is MISSING:
            return [("DEAD", path)]
        if isinstance(node, dict) and node:
            return [f for k, v in node.items() for f in ignored(v, path + (k,))]
        return [("CONDITIONAL", path)]

    def walk(node, path):
        for k, child in node.items():
            p = path + (k,)
            if r.render(without(values, p)) == baseline:
                d = lookup(defaults, p)
                if matches_default(child, d) or r.render(replaced(values, p, mutate(child))) != baseline:
                    findings.append(("REDUNDANT", p))
                else:
                    findings.extend(ignored(child, p))
            elif isinstance(child, dict) and child:
                walk(child, p)

    walk(values, ())
    return t, findings, None


def changed_files(ref):
    out = subprocess.run(["git", "diff", "--name-only", f"{ref}...HEAD"], capture_output=True, text=True, check=True)
    return set(out.stdout.split())


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--changed-since")
    ap.add_argument("--show-redundant", action="store_true")
    ap.add_argument("--cache", default=os.path.join(tempfile.gettempdir(), "helm-dead-values"))
    args = ap.parse_args()

    k3s = k3s_version()
    with open(API_VERSIONS_FILE) as f:
        api_versions = [l.strip() for l in f if l.strip()]
    helm_args = ["--kube-version", k3s.split("+")[0]]
    for a in api_versions:
        helm_args += ["--api-versions", a]

    targets = argocd_targets() + [traefik_target(k3s)]
    if args.changed_since:
        changed = changed_files(args.changed_since)
        if not changed & TOOL_FILES:
            targets = [t for t in targets if changed & {t.source_file, *t.values_files}]
    if not targets:
        print("no Helm values changed")
        return

    gha = os.environ.get("GITHUB_ACTIONS") == "true"
    failed = False
    with ThreadPoolExecutor(max_workers=os.cpu_count() or 2) as pool:
        results = list(pool.map(lambda t: analyze(t, args.cache, helm_args), targets))
    for t, findings, err in results:
        where = t.values_files[0] if t.values_files else t.source_file
        if err:
            failed = True
            print(f"::error file={where}::{t.name}: {err}" if gha else f"ERROR {t.name}: {err}")
            continue
        shown = [(k, p) for k, p in findings if k in FAILING or args.show_redundant]
        print(f"{t.name:28} {t.chart} {t.version}: " + (", ".join(f"{k} {'.'.join(map(str, p))}" for k, p in shown) or "ok"))
        for kind, p in shown:
            if kind in FAILING:
                failed = True
                if gha:
                    print(f"::error file={where},title={kind} Helm value::{t.name}: {'.'.join(map(str, p))} has no effect on the rendered chart")
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()
