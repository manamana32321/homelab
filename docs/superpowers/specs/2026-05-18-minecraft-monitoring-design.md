# Minecraft 모니터링 + 운영 알림

> Issue: [manamana32321/homelab#206](https://github.com/manamana32321/homelab/issues/206)
> Date: 2026-05-18
> 도구 선정 변경 근거: README의 [UnifiedMetrics 권고](../../../k8s/minecraft/README.md)는 폐기 — 실측 결과 유지보수 정체(정식 릴리스 v0.3.8 = 2023-04, Paper 1.21.x 미지원).

## 배경

마인크래프트 서버에 운영 모니터링이 없다. 백업([#205](https://github.com/manamana32321/homelab/issues/205))이 적용된 상태에서 다음 안전망은:
1. **서버 가용성**: 다운 시 즉시 알람
2. **인게임 메트릭**: 플레이어 수, TPS, 메모리 — Grafana로 가시화
3. **Pod 리소스**: CPU/메모리 한계 근접 시 알람

기존 홈랩 Prometheus/Grafana/Alertmanager 스택에 합류 — 별도 모니터링 인프라 안 띄움.

## 도구 선정 변경 — UnifiedMetrics → sladkoff

| | UnifiedMetrics | sladkoff/minecraft-prometheus-exporter |
|---|---|---|
| ★ | (확인 못 함) | 530 |
| 마지막 정식 | v0.3.8 (2023-04) | v3.1.2 (2025-02) |
| 최근 활동 | SNAPSHOT pre-release만 (2025-04) | 2026-05-09 push |
| Paper 지원 | 1.8 ~ 1.19.4 (Hangar 핀) | api-version 1.16, 1.21 issue 종결 |
| 1.21.x 검증 | ❌ 미지원 | ✅ 호환 (open issue 0개) |
| 메트릭 endpoint | port 9100/9970 (불명확) | port `9940`, `/metrics` (README 명시) |

**결정**: sladkoff 채택. UnifiedMetrics는 폐기 (생태계 정체).

## 설계 개요

```
            ┌──────────────────────────────────────────────┐
            │  Minecraft Pod (json-server-1)               │
            │  ┌────────────────────┐  ┌──────────────────┐│
            │  │ minecraft 컨테이너 │  │ mc-backup 사이드카│
            │  │  + PrometheusExporter│ │ (이미 배포됨)    ││
            │  │  plugin :9940      │  │                  ││
            │  └────────────────────┘  └──────────────────┘│
            └─────────────┬────────────────────────────────┘
                          │ /metrics (port 9940)
        ┌─────────────────┴─────────────────┐
        ▼                                   ▼
  Service: minecraft-metrics       Blackbox Exporter (TCP)
  + ServiceMonitor                 probes minecraft:25565
        │                                   │
        └─────────► Prometheus ◄────────────┘
                       │
              ┌────────┴────────┐
              ▼                 ▼
          Grafana          Alertmanager
          (대시보드)        → Telegram (기존)
```

## 구성 요소

### 1. 인게임 exporter — sladkoff 플러그인

**설치** (`values.yaml`):
- `pluginUrls`에 [v3.1.2 jar URL](https://github.com/sladkoff/minecraft-prometheus-exporter/releases/download/v3.1.2/minecraft-prometheus-exporter-3.1.2.jar) 추가
- `extraEnv.MINECRAFT_PROMETHEUS_EXPORTER_HOST: "0.0.0.0"` — config.yml 안 건드리고 env로 바인드 호스트 오버라이드 (플러그인 README 명시 지원)

**노출** (`k8s/minecraft/manifests/service-metrics.yaml` 신규):
- ClusterIP Service, port 9940, selector `app: minecraft`
- 별도 Service로 분리 — 메인 LoadBalancer Service(25565)에 메트릭 포트 섞지 않음

**스크레이프** (`k8s/minecraft/manifests/service-monitor.yaml` 신규):
- ServiceMonitor, label `release: prometheus` (kube-prometheus-stack discovery)
- 30s interval (사이즈 작은 메트릭이라 빈도 적당)

**노출 메트릭** (주요):
- `mc_players_online_total`, `mc_players_total`
- `mc_tps`, `mc_tick_duration_average`, `mc_tick_duration_max`
- `mc_jvm_memory_used_bytes`, `mc_jvm_gc_*`
- `mc_entities_total`, `mc_loaded_chunks_total`

### 2. 가용성 프로브 — Blackbox TCP

**대상**: `minecraft.minecraft.svc.cluster.local:25565` (클러스터 내부 probe)

**구현**: `k8s/observability/blackbox-exporter/values.yaml`의 `targets`에 추가
- 모듈: `tcp_connect` (HTTP 아님)
- 외부 coral 프로브는 추가 안 함 — 친구 서버 규모에 과함, 내부 ClusterIP 프로브가 "서비스 다운"을 충분히 잡음

### 3. PrometheusRule (`k8s/minecraft/manifests/prometheus-rule.yaml` 신규)

새 PrometheusRule CR — `minecraft` 그룹. label `release: prometheus` 필수.

| 알림 | 조건 | severity | 의미 |
|---|---|---|---|
| `MinecraftServerDown` | `probe_success{job=~".*blackbox.*",instance=~".*minecraft.*"} == 0` for 2m | critical | TCP 연결 실패 = 서버 다운 |
| `MinecraftLowTPS` | `mc_tps < 18` for 5m | warning | TPS 저하 (정상 ~20) |
| `MinecraftHighMemoryUsage` | `container_memory_working_set_bytes / container_spec_memory_limit_bytes > 0.9` for 5m | warning | OOMKill 임박 |
| `MinecraftPodRestartLoop` | `increase(kube_pod_container_status_restarts_total[15m]) > 3` | critical | 잦은 재시작 |

PVC 가득함은 기존 `PVCAlmostFull` 글로벌 룰이 잡으므로 추가 불요.

### 4. Grafana 대시보드 (`k8s/observability/grafana/minecraft-dashboard.yaml` 신규)

sladkoff 공식 대시보드 [minecraft-server-dashboard.json](https://github.com/sladkoff/minecraft-prometheus-exporter/blob/master/dashboards/minecraft-server-dashboard.json)을 ConfigMap으로 감쌈.
- label `grafana_dashboard: "1"` → kube-prometheus-stack Grafana sidecar가 자동 import
- annotation `grafana_folder: Apps` (health-hub와 동일 폴더)

## 결정 필요

### 결정 1 — 인게임 exporter 채택 여부

- **권고: 채택 (sladkoff v3.1.2)** — 1.21.11 호환 신뢰 가능 (api-version 1.16, open 1.21 issue 0)
- 대안: skip — Blackbox + Pod 리소스 알람만으로 1차 모니터링 (TPS·플레이어 수 알람 없음)

### 결정 2 — Blackbox 프로브 범위

- **권고: 클러스터 내부 1개** (`minecraft.minecraft.svc:25565` via tcp_connect)
- 대안: 외부 coral 프로브도 추가 — DNS/공유기/ufw까지 검증, 친구 서버 규모엔 과함

### 결정 3 — Grafana 대시보드 import

- **권고: sladkoff 공식 대시보드 ConfigMap 1개 import**
- 대안: skip — 메트릭은 Prometheus에 있지만 Grafana 가시화는 없음 (`/explore`로 쿼리만 가능)

### 결정 4 — Alert 임계치

- ServerDown 2m: 기존 EndpointDown과 동일 (timeline)
- LowTPS < 18 for 5m: 정상 ~20에서 10% 손실, 친구 몇 명 규모에 적정 (수치 조정 가능)
- HighMemory > 90% for 5m: OOMKill 직전 (수치 조정 가능)

## 구현 단계 (#206 acceptance)

1. [ ] `k8s/minecraft/values.yaml` — `pluginUrls` + `extraEnv` 추가
2. [ ] `k8s/minecraft/manifests/service-metrics.yaml` 신규 (Service)
3. [ ] `k8s/minecraft/manifests/service-monitor.yaml` 신규 (ServiceMonitor)
4. [ ] `k8s/minecraft/manifests/prometheus-rule.yaml` 신규 (PrometheusRule)
5. [ ] `k8s/observability/blackbox-exporter/values.yaml` — minecraft TCP target 추가
6. [ ] `k8s/observability/grafana/minecraft-dashboard.yaml` 신규 (ConfigMap)
7. [ ] `helm template` 사전 검증 — extraEnv·pluginUrls 반영 확인 (#213 사고 교훈)
8. [ ] PR → 머지 → ArgoCD apps + minecraft + prometheus + grafana 4개 동기 (app-of-apps 2-hop)
9. [ ] 검증: Pod 로그에서 "Started Prometheus metrics endpoint at: 0.0.0.0:9940"
10. [ ] ServiceMonitor 스크레이프 확인 (`up{job="minecraft-metrics"} == 1`)
11. [ ] Blackbox 프로브 확인 (`probe_success{instance=~".*minecraft.*"} == 1`)
12. [ ] Grafana에서 대시보드 표시 확인
13. [ ] Alertmanager에 알림 룰 4개 등록 확인 (트리거 테스트는 선택)

## 리스크

- **플러그인 1.21.11 동작 실패**: 가능성 낮음(api-version 1.16 + 1.21 issue 종결) but 사전 검증 불가능 (helm template은 컨테이너 시작 후 jar 다운로드 단계를 못 봄). 머지 후 Pod 로그가 "Started Prometheus metrics endpoint" 안 보이면 폴백 결정 필요
- **app-of-apps 2-hop**: ArgoCD Application은 안 건드리므로 이번엔 1-hop만, but values.yaml 변경분이 apps 거치지 않아서 minecraft만 sync하면 됨. 단 prometheus/grafana 변경은 각 앱 sync 필요 (3 hops 총)
- **메트릭 이름 충돌**: sladkoff 메트릭 prefix가 `mc_` — 다른 노출의 `minecraft_*` 등과 안 겹침
