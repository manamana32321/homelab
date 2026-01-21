# Observability Stack

메트릭, 로그, 트레이스를 수집하고 시각화합니다.

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

| 컴포넌트 | 용도 |
|----------|------|
| OTel Collector | OTLP 수신, 메트릭/로그/트레이스 분배 |
| Prometheus | 메트릭 저장 |
| Loki | 로그 저장 |
| Tempo | 트레이스 저장 |
| Promtail | 노드 로그 수집 → Loki |
| Grafana | 대시보드 시각화 |

## 접속 방법

| 서비스 | URL | 인증 |
|--------|-----|------|
| Grafana | `grafana.json-server.win` | - |
| Prometheus | `prometheus.json-server.win` | - |

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
```
