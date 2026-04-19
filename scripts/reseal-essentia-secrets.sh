#!/usr/bin/env bash
# Re-seal saemate ns SealedSecret values into essentia ns scope.
#
# Why: SealedSecret default scope is `strict` — moving a secret to a new
# namespace requires re-sealing because the encryption is bound to
# `<namespace>/<name>`.
#
# Prereqs:
#   - kubectl context pointing to the homelab cluster (context: json)
#   - saemate namespace still exists with saemate-api-secrets + ghcr-saemate
#     live Secret objects (controller-decrypted)
#   - Repo root checked out, pwd = repo root
#   - KUBECONFIG=~/.kube/config-<cluster> set (per user's CLAUDE.md rule)
#
# Output: updates k8s/essentia/api/sealed-secret.yaml and
#         k8s/essentia/ghcr-pull-sealed-secret.yaml in-place.

set -euo pipefail

CERT="k8s/sealed-secrets/cert.pem"
API_SEALED="k8s/essentia/api/sealed-secret.yaml"
GHCR_SEALED="k8s/essentia/ghcr-pull-sealed-secret.yaml"
MINIO_SEALED="k8s/minio-tenants/overlays/essentia/sealed-secret.yaml"

if [[ ! -f "$CERT" ]]; then
  echo "ERROR: $CERT not found. Run from repo root." >&2
  exit 1
fi

if [[ -z "${KUBECONFIG:-}" ]]; then
  echo "ERROR: KUBECONFIG must be set (per CLAUDE.md kubeseal-guard rule)." >&2
  echo "Example: KUBECONFIG=~/.kube/config-json ./scripts/reseal-essentia-secrets.sh" >&2
  exit 1
fi

echo "▶ Fetching saemate-api-secrets plaintext from cluster..."
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

kubectl -n saemate get secret saemate-api-secrets -o json \
  | jq -r '.data | to_entries[] | "\(.key)=\(.value | @base64d)"' \
  > "$TMPDIR/api-secrets.env"

echo "▶ Fetching ghcr-saemate plaintext from cluster..."
kubectl -n saemate get secret ghcr-saemate -o json \
  | jq -r '.data[".dockerconfigjson"] | @base64d' \
  > "$TMPDIR/dockerconfig.json"

echo "▶ Building new SealedSecret manifest for essentia-api-secrets..."
# Build a plaintext Secret manifest for essentia ns, then kubeseal into place.
{
  echo "apiVersion: v1"
  echo "kind: Secret"
  echo "metadata:"
  echo "  name: essentia-api-secrets"
  echo "  namespace: essentia"
  echo "type: Opaque"
  echo "stringData:"
  while IFS='=' read -r k v; do
    # escape any trailing whitespace / control chars into YAML block literal for safety
    printf "  %s: |-\n    %s\n" "$k" "$v"
  done < "$TMPDIR/api-secrets.env"
} > "$TMPDIR/essentia-api-secrets.yaml"

kubeseal --format=yaml --cert "$CERT" \
  --scope strict \
  < "$TMPDIR/essentia-api-secrets.yaml" \
  > "$API_SEALED"

echo "▶ Building new SealedSecret manifest for ghcr-essentia..."
kubectl create secret docker-registry ghcr-essentia \
  --namespace=essentia \
  --docker-server=ghcr.io \
  --docker-username="$(jq -r '.auths["ghcr.io"].username' "$TMPDIR/dockerconfig.json")" \
  --docker-password="$(jq -r '.auths["ghcr.io"].auth' "$TMPDIR/dockerconfig.json" | base64 -d | cut -d: -f2-)" \
  --docker-email=noreply@example.com \
  --dry-run=client -o yaml \
  | kubeseal --format=yaml --cert "$CERT" --scope strict \
  > "$GHCR_SEALED"

echo "▶ Re-sealing minio-config..."
MINIO_CONFIG_ENV=$(kubectl -n saemate get secret minio-config -o json \
  | jq -r '.data["config.env"] | @base64d')

{
  echo "apiVersion: v1"
  echo "kind: Secret"
  echo "metadata:"
  echo "  name: minio-config"
  echo "  namespace: essentia"
  echo "type: Opaque"
  echo "stringData:"
  printf "  config.env: |-\n"
  printf '%s\n' "$MINIO_CONFIG_ENV" | sed 's/^/    /'
} > "$TMPDIR/minio-config.yaml"

kubeseal --format=yaml --cert "$CERT" --scope strict \
  < "$TMPDIR/minio-config.yaml" \
  > "$MINIO_SEALED"

echo "✅ Re-sealed:"
echo "   - $API_SEALED"
echo "   - $GHCR_SEALED"
echo "   - $MINIO_SEALED"
echo
echo "Next: review diff, commit, push, merge. ArgoCD syncs on merge."
