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
- `resources`, `coral`, `crowdsec` — 환경 특화

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
                    ┌───────────────────────┐
                    │  OpenTelemetry        │
      App ─────────▶│  Collector            │
     (OTLP)         │ (텔레메트리 수집/변환)  │
                    └───────────┬───────────┘
                                │
       ┌────────────────────────┼────────────────────────┐
       ▼                        ▼                        ▼
┌─────────────┐          ┌─────────────┐          ┌─────────────┐
│  Prometheus │          │    Loki     │          │    Tempo    │
│  (메트릭)    │          │   (로그)    │          │  (트레이스)  │
└──────┬──────┘          └──────┬──────┘          └──────┬──────┘
       │                        ▲                        │
       │                        │                        │
       │                 ┌──────┴──────┐                 │
       │                 │  Promtail   │                 │
       │                 │ (노드 로그)  │                 │
       │                 └─────────────┘                 │
       │                                                 │
       └────────────────────────┬────────────────────────┘
                                │
                         ┌──────▼──────┐
                         │   Grafana   │
                         │  (대시보드)  │
                         └─────────────┘
```

## 컴포넌트

| 컴포넌트 | 용도 | 룰 출처 |
|---|---|---|
| OTel Collector | OTLP 수신, 메트릭/로그/트레이스 분배 | - |
| Prometheus (kube-prometheus-stack) | 메트릭 저장 + Alertmanager + 기본 알림 룰 | chart |
| Loki | 로그 저장 | (없음) |
| Tempo | 트레이스 저장 | (없음) |
| Promtail | 노드 로그 수집 → Loki | (없음) |
| Grafana | 대시보드 시각화 (알림 미사용) | - |
| Blackbox Exporter | HTTP/TCP probe | `homelab-alerts` (도메인 특화) |
| smartctl-exporter | SMART 디스크 건강 | chart + `homelab-alerts` |

## 알림 채널

```
PrometheusRule → Prometheus → Alertmanager → Telegram (chat -1003865568684, thread 206)
```

- **Alertmanager 설정**: [prometheus/values.yaml](prometheus/values.yaml) `alertmanager.config`
- **bot token**: SealedSecret `alertmanager-telegram` (observability ns)
- **Watchdog/InfoInhibitor**: `null` receiver로 명시적 무시 (kube-prometheus-stack 기본)
- **Grafana Unified Alerting 미사용** — 단일 notification path 유지 (단일 사고 경로 = 디버깅 용이)

## 접속

| 서비스 | URL |
|---|---|
| Grafana | grafana.json-server.win |
| Prometheus | prometheus.json-server.win |
| Alertmanager | (외부 노출 안 함, kubectl port-forward) |

## 앱에서 텔레메트리 전송

### 클러스터 내부

```yaml
env:
  - name: OTEL_EXPORTER_OTLP_ENDPOINT
    value: "http://otel-collector-opentelemetry-collector.observability.svc.cluster.local:4317"
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

# OTel Collector 로그
kubectl logs -n observability -l app.kubernetes.io/name=opentelemetry-collector

# Alertmanager 큐 상태
kubectl port-forward -n observability svc/prometheus-kube-prometheus-alertmanager 9093:9093
# http://localhost:9093
```
