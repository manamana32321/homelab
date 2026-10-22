# k8s/observability/ — Claude Code Guidance

> 메트릭·로그·트레이스·알림 스택의 **선언적 SSOT**.
> 신규 알림 룰 작성 전 반드시 "알림 룰 정책" 섹션 확인.

## 알림 룰 작성 정책

> **재발명 금지. 기존 룰 사용이 기본, 직접 정의는 최후 수단.**

### 검색 우선순위

신규 알림이 필요하면 다음 순서로 기존 자산을 먼저 확인:

1. **kube-prometheus-stack 기본 룰** — 클러스터 일반 사고 (Pod/Node/PVC/Job/Target/Apiserver 등)
   ```bash
   kubectl get prometheusrules -n observability | grep prometheus-kube-prometheus
   ```
   대표 룰: `TargetDown` (scrape target up==0 → 컨트롤러·exporter 다운 자동 catch), `KubePodCrashLooping`, `KubeJobFailed`, `NodeFilesystemSpaceFillingUp`, `KubePersistentVolumeFillingUp`

2. **컴포넌트 chart 내장 룰** — 해당 Helm chart의 `prometheusRules.rules` 또는 `metrics.rules.spec` slot
   - 예: [argo-cd chart](https://github.com/argoproj/argo-helm/tree/main/charts/argo-cd) `controller.metrics.rules.spec` (ArgoAppMissing / ArgoAppNotSynced commented examples)
   - 예: [smartctl-exporter chart](https://github.com/prometheus-community/helm-charts/tree/main/charts/prometheus-smartctl-exporter) `prometheusRules.rules` (Temperature, Status 등)
   - chart 내장이면 chart values에서 enable + threshold만 조정하고 끝

3. **vetted community 출처**
   - [samber/awesome-prometheus-alerts](https://samber.github.io/awesome-prometheus-alerts/) — 200+ 컴포넌트 curated. 그대로 import 시 noisy expr 검수 필요
   - PromQL은 동일하니 wording만 맞춰서 차용

4. **직접 정의** — 위 셋 중 어디에도 없을 때만

### 직접 정의 위치

[k8s/observability/prometheus/manifests/alert-rules.yaml](prometheus/manifests/alert-rules.yaml)의 `homelab-alerts` PrometheusRule. 그룹별로 분리해서 작성:
- `blackbox` — Blackbox Exporter probe (kube-prometheus-stack 기본에 없음)
- `storage`, `smart` — PVC/디스크 임계치, ATA 어트리뷰트 (도메인 특화)
- `resources`, `crowdsec` — 환경 특화

### annotation 작성

Alertmanager 디스코드 템플릿이 `summary`와 `description`을 연달아 출력한다. 둘이 같은 값을 반복하면 메시지 길이가 두 배가 된다.

| annotation | 담는 것 | 금지 |
|---|---|---|
| `summary` | 알림 종류 한 줄 | 라벨 값 — 식별은 description이 한다 |
| `description` | 대상 식별 + 관측 수치 | summary 반복, 원인 추정, 조치 절차 |

```yaml
- alert: EndpointDown
  annotations:
    summary: "엔드포인트 응답 없음"
    description: "{{ $labels.instance }} ({{ $labels.job }})이 2분간 probe 실패."
```

- 한국어로 쓴다
- 룰은 actionable해야 하지만(아래 Anti-patterns) **절차 자체는 본문에 쓰지 않는다**. 알림은 "무엇이 관측됐나"까지
- `runbook_url`은 현재 디스코드 템플릿이 렌더하지 않으므로 쓰지 않는다

**선례**: [PR #335](https://github.com/manamana32321/homelab/pull/335)에서 24건 전수 정리. 설명 평균 144자 → 79자, summary 라벨 중복 16건 → 0건.

### chart 내장 룰이 broken인 경우

chart 룰이 메트릭명 mismatch 같은 버그를 가지면:

1. chart values에서 해당 룰만 `enabled: false`로 명시 (이유 주석)
2. `homelab-alerts`에 같은 의도의 룰을 **정확한 메트릭명**으로 작성
3. 주석에 "chart 버그로 대체"와 chart 룰명 명시

**선례**: smartctl-exporter chart의 `smartCTLDeviceStatus`는 expr이 `smartctl_device_status`(미존재) 참조 → 실제 메트릭은 `smartctl_device_smart_status`. → [PR #192](https://github.com/manamana32321/homelab/pull/192)에서 chart 룰 disable + `homelab-alerts.smart.DiskSmartFailed`로 대체.

### Anti-patterns

- ❌ chart에 동일 의도 룰 있는데 `homelab-alerts`에 중복 작성
- ❌ kube-prometheus-stack 일반 룰(`TargetDown` 등)을 컴포넌트별 specific 알림으로 재정의 — `up{namespace="argocd"} == 0` 같은 룰은 이미 `TargetDown`이 catch
- ❌ "혹시 모르니까" 룰 무더기 추가 — 각 룰은 **운영상 actionable**해야 함 (받는 사람이 즉시 행동할 수 있어야 함)
- ❌ NVMe 전용 메트릭 룰을 SATA-only 환경에서 enable — 발화 불가능한 dead rule
- ❌ description에 조치 절차(`kubectl ...`)나 원인 추정 서술 — 28종이 동시에 울 때 긴 알림은 안 읽힌다

### 룰 전수 검증

알림 시스템 변경 후엔 항상 다음 확인:

```bash
# 모든 룰이 healthy
kubectl exec prometheus-prometheus-kube-prometheus-prometheus-0 -c prometheus -n observability \
  -- wget -qO- http://localhost:9090/api/v1/rules | \
  jq '.data.groups[].rules[] | select(.health!="ok")'

# 의도하지 않은 firing 알림 없는지
kubectl exec prometheus-prometheus-kube-prometheus-prometheus-0 -c prometheus -n observability \
  -- wget -qO- http://localhost:9090/api/v1/alerts | \
  jq '.data.alerts[] | {state, alertname: .labels.alertname}'
```

---

## 아키텍처

```
 App OTLP ───────┐
 OpenWrt syslog ─┼──▶ alloy-gateway (Deployment) ──┬──▶ Prometheus (OTLP, remote write)
 k8s Event ──────┘    blackbox probes              ├──▶ Loki
                                                   └──▶ Tempo

 Pod logs ───────┐
 journald ───────┤
 containerd ─────┼──▶ alloy-node (DaemonSet) ──▶ Loki
 apt, dpkg ──────┘    └ prometheus.exporter.unix ◀── Prometheus scrape

 ServiceMonitor targets ◀── Prometheus scrape
 Prometheus, Loki, Tempo ──▶ Grafana
```

## 컴포넌트

| 컴포넌트 | 용도 | 룰 출처 |
|---|---|---|
| alloy-node (DaemonSet) | 파드 로그·journald·containerd·apt·dpkg → Loki, 호스트 메트릭 노출 (`job="node-exporter"`) | chart (node-exporter 룰) |
| alloy-gateway (Deployment) | OTLP 수신, OpenWrt syslog, blackbox 프로브, k8s Event → Loki | `homelab-alerts` (blackbox) |
| Prometheus (kube-prometheus-stack) | 메트릭 저장 + Alertmanager + 기본 알림 룰 | chart |
| Loki | 로그 저장 | (없음) |
| Tempo | 트레이스 저장 | (없음) |
| Grafana | 대시보드 시각화 (알림 미사용) | - |
| smartctl-exporter | SMART 디스크 건강 | chart + `homelab-alerts` |

### Loki 라벨

| source | 라벨 |
|---|---|
| `pods` | `app component container filename instance job namespace node node_name pod stream` |
| `journal` | `node unit level` (`avahi-daemon.service` 제외) |
| `containerd` | `node level` |
| `apt`, `dpkg` | `node` |
| `openwrt` | `service_name=openwrt node=ax3000sm unit level` |
| `k8s-events` | `namespace kind reason type` |

### blackbox 대상

[alloy-gateway/values.yaml](alloy-gateway/values.yaml) `prometheus.exporter.blackbox "probes"` 의 `targets`. 메트릭 라벨 `job="blackbox"`, `instance=<주소>`, `target=<이름>`.

## 알림 채널

```
PrometheusRule → Prometheus → Alertmanager → Discord (#alerts 채널 웹훅)
```

- **Alertmanager 설정**: [prometheus/values.yaml](prometheus/values.yaml) `alertmanager.config`
- **webhook URL**: SealedSecret `alertmanager-discord` (observability ns)
- **Watchdog/InfoInhibitor**: `null` receiver로 명시적 무시 (kube-prometheus-stack 기본)
- **Grafana Unified Alerting 미사용** — 단일 notification path 유지 (단일 사고 경로 = 디버깅 용이)

## 접속

| 서비스 | URL |
|---|---|
| Grafana | grafana.json-server.win |
| Prometheus | (외부 노출 안 함, kubectl port-forward) |
| Alertmanager | (외부 노출 안 함, kubectl port-forward) |

Prometheus·Alertmanager는 자체 인증 기능이 없어 ingress가 곧 무인증 공개 API가 된다.
둘 다 ingress를 두지 않고 port-forward로만 접근한다. 메트릭 조회는 Grafana를 쓴다.

## 앱에서 텔레메트리 전송

### 클러스터 내부

```yaml
env:
  - name: POD_NAME
    valueFrom:
      fieldRef:
        fieldPath: metadata.name
  - name: OTEL_EXPORTER_OTLP_ENDPOINT
    value: "http://alloy-gateway.observability.svc.cluster.local:4318"
  - name: OTEL_RESOURCE_ATTRIBUTES
    value: "service.namespace=<프로젝트>,deployment.environment.name=<환경>,service.instance.id=$(POD_NAME)"
```

### 클러스터 외부

```yaml
env:
  - name: OTEL_EXPORTER_OTLP_ENDPOINT
    value: "https://otel.json-server.win"
```

## 트러블슈팅

```bash
# Prometheus 타겟 상태
kubectl port-forward -n observability svc/prometheus-kube-prometheus-prometheus 9090:9090
# http://localhost:9090/targets

# Loki 상태
kubectl logs -n observability -l app.kubernetes.io/name=loki

# Alloy 로그 (UI: kubectl port-forward -n observability svc/alloy-gateway 12345)
kubectl logs -n observability -l app.kubernetes.io/instance=alloy-gateway -c alloy
kubectl logs -n observability -l app.kubernetes.io/instance=alloy-node -c alloy

# Alertmanager 큐 상태
kubectl port-forward -n observability svc/prometheus-kube-prometheus-alertmanager 9093:9093
# http://localhost:9093
```
