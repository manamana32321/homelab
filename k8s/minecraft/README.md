# Minecraft 서버

홈랩 K3s에 배포된 마인크래프트 서버와 운영 도구 가이드.

- **주소**: `mc.json-server.win` (Java Edition, 정품 계정 전용)
- **차트**: [itzg/minecraft-server](https://github.com/itzg/minecraft-server-charts) Helm chart `5.0.0`
- **게임 버전**: Paper `1.21.11` — [values.yaml](values.yaml)
- **ArgoCD**: [k8s/argocd/applications/apps/minecraft/](../argocd/applications/apps/minecraft/)

## 현재 배포 상태

| 구성 | 상태 |
|---|---|
| 서버 (Paper 1.21.11) | ✅ 배포됨 |
| LoadBalancer Service (k3s ServiceLB, 25565/TCP) | ✅ 배포됨 |
| 운영 도구 6종 | ⬜ 미배포 — 본 문서가 도입 가이드 |

## ⚠️ 게임 버전 제약 — `1.21.11` 핀 유지

Mojang이 2026년 연도 기반 버전 체계(`26.x`)로 전환하면서 Paper 26.1이 서버 jar 난독화/remapper를 제거 → **플러그인 ABI 단절**. CoreProtect 등 핵심 플러그인의 26.1+ 정식 빌드가 아직 미비하다. `1.21.11`(2025-12)이 마지막 안정 1.x이며 26.1.2와 달력상 4개월 차이뿐이다.

**도구 도입 시 반드시 1.21.11 호환 빌드로 확인할 것.** 생태계가 26.x 정식 빌드를 내면 그때 `minecraftServer.version` 한 줄로 롤포워드.

---

## 운영 도구 6개 카테고리

> 출처: tech-scout 2회 조사 (운영 도구 생태계 / batteries-included 차트 검토).
> **핵심 결론**: 별도 "올인원" 차트는 존재하지 않으며 불필요 — itzg 차트가 백업·플러그인·RCON을 values로 이미 내장한다. 나머지 3개(모니터링·웹맵·알림)만 차트 외부에서 조합하며, 전부 기존 홈랩 인프라를 재사용한다.

### 1. 월드 백업

| | |
|---|---|
| **무엇** | 월드 데이터 자동 백업 — 일관성 보장, 오프사이트 저장, 복원 |
| **후보** | itzg/mc-backup 사이드카 / K8s CronJob 자체조립 / Velero / Longhorn 스냅샷 |
| **권고** | **itzg/mc-backup 사이드카 + restic → AWS S3** |
| **이유** | itzg 생태계 네이티브, RCON으로 `save-off`/`save-on` 조율해 일관성 보장, restic이 기존 S3 백업 인프라(tfstate·Seafile)에 합류, 사이드카라 순수 선언적 |
| **차트 내장** | ✅ `mcbackup.*` (values.yaml에서 설정) |
| **홈랩 통합** | restic repo 비밀번호 + S3 키 → SealedSecret 2개. 스케줄은 새벽 KST |

### 2. 모니터링 / 메트릭

| | |
|---|---|
| **무엇** | player count, TPS, tick time, 메모리 등을 Prometheus로 노출 |
| **후보** | UnifiedMetrics / sladkoff exporter / mineGrafana / ServerPulse |
| **권고** | **UnifiedMetrics** (Prometheus exporter 플러그인) |
| **이유** | 순수 exporter라 기존 Prometheus 스택에 ServiceMonitor 하나로 직결, 별도 스택 안 띄움, 플러그인이라 `MODRINTH_PROJECTS`로 선언적 설치 |
| **차트 내장** | ❌ 외부 (플러그인) |
| **홈랩 통합** | 기존 Prometheus + Grafana 재사용. 배포 시 26.x 아닌 **1.21.11 호환 빌드 확인 필수** (sladkoff는 유지보수 정체라 비채택) |

### 3. 필수 플러그인

| | |
|---|---|
| **무엇** | 운영 기본기 — 권한 관리, 그리핑 롤백, 지역 보호, 코어 관리 명령 |
| **권고** | **EssentialsX**(코어 명령) · **LuckPerms**(권한) · **CoreProtect**(그리핑 롤백) · **WorldGuard**(지역 보호) |
| **이유** | 영역별 사실상 표준 — 모든 가이드가 동일 4종 지목. 화이트리스트/밴은 `onlineMode:true` + 소규모라 바닐라 기능으로 충분 (별도 플러그인 불필요) |
| **차트 내장** | ✅ `modrinth.projects` / `pluginUrls` (values.yaml에서 선언 → 차트가 SSOT) |
| **홈랩 통합** | EssentialsX·LuckPerms·CoreProtect는 Modrinth 배포 → `MODRINTH_PROJECTS`. WorldGuard는 Modrinth 미배포 → `PLUGINS` URL. **CoreProtect 1.21.11 정식 빌드 존재 확인 필수** |

### 4. 웹 맵 뷰어

| | |
|---|---|
| **무엇** | 월드를 브라우저에서 보는 맵 (`map.minecraft.json-server.win` 등) |
| **후보** | BlueMap / Dynmap / squaremap |
| **권고** | **BlueMap** — standalone 모드로 **별도 Deployment 분리 배포** |
| **이유** | standalone CLI 모드가 있어 마크 서버 Deployment와 물리적으로 분리 가능 → 맵 렌더 부하와 게임 틱 부하 격리. K8s에서 가장 깔끔한 패턴 |
| **차트 내장** | ❌ 외부 (별도 Deployment + Ingress) |
| **홈랩 통합** | 월드 PVC를 read-only 마운트하는 별도 Deployment, Ingress + cert-manager DNS01. 소규모면 일단 플러그인 모드로 시작하고 부하 보이면 분리해도 됨 |

### 5. 원격 관리 / 콘솔

| | |
|---|---|
| **무엇** | 콘솔 명령 실행, start/stop, 디버깅 |
| **권고** | **RCON + Loki 로그 + ArgoCD** — 별도 도구 채택 안 함 |
| **이유** | itzg 이미지에 `rcon-cli` 내장(`kubectl exec`), 로그는 기존 Loki로 자동 수집, start/stop은 `kubectl scale`/ArgoCD. **웹 패널(Pterodactyl/Crafty)은 비채택** — 자체 오케스트레이터를 내장해 ArgoCD와 컨트롤 플레인이 이중화됨 (이중 SSOT 안티패턴) |
| **차트 내장** | ✅ RCON (`minecraftServer.rcon.*`) |
| **홈랩 통합** | RCON 비밀번호 → SealedSecret. RCON 포트는 LoadBalancer로 노출하지 말고 `kubectl port-forward`로만 |

### 6. 알림 / 이벤트 통합

운영 알림과 게임플레이 이벤트를 **분리**하는 게 핵심.

| 종류 | 권고 | 이유 |
|---|---|---|
| **운영 알림** (서버 다운, TPS 저하) | **Prometheus Alertmanager** (기존 인프라 재사용) | 카테고리 2의 exporter + Blackbox Exporter TCP 프로빙 → 기존 Telegram(thread 206) 파이프라인. 새 도구 0개 |
| **게임플레이 이벤트** (접속/사망/채팅) | **DiscordSRV** (Discord 채널 신설) | 카테고리 독점 표준, 양방향 채팅. 운영 노이즈와 게임 노이즈 채널 분리가 올바른 설계 |

---

## itzg 차트 내장 vs 외부 — 한눈에

| 카테고리 | 차트 내장? | 어떻게 |
|---|---|---|
| 백업 | ✅ | `mcbackup.*` |
| 플러그인 | ✅ | `modrinth.projects` / `pluginUrls` |
| RCON | ✅ | `minecraftServer.rcon.*` |
| 모니터링 | ❌ | UnifiedMetrics 플러그인 + ServiceMonitor |
| 웹맵 | ❌ | BlueMap standalone Deployment + Ingress |
| 알림 | ❌ | 기존 Alertmanager + DiscordSRV 플러그인 |

---

## 도입 순서 제안

운영 안정성 기여도 순. 각 단계는 별도 이슈 + PR.

1. **백업** ([#205](https://github.com/manamana32321/homelab/issues/205)) — 데이터 보호가 최우선. 다른 작업의 안전망. (차트 내장이라 PR 1개)
2. **모니터링 + 운영 알림** ([#206](https://github.com/manamana32321/homelab/issues/206)) — 서버 상태 가시성. 이후 작업의 검증 수단도 됨.
3. **필수 플러그인** ([#207](https://github.com/manamana32321/homelab/issues/207)) — EssentialsX·LuckPerms·CoreProtect·WorldGuard. 그리핑 롤백은 친구 서버에서도 사실상 필수.
4. **웹맵** ([#208](https://github.com/manamana32321/homelab/issues/208)) — 있으면 좋은 것. 부하 격리 위해 별도 Deployment.
5. **DiscordSRV** ([#209](https://github.com/manamana32321/homelab/issues/209)) — 게임플레이 이벤트. Discord 서버 신설 전제.

---

## 참고

- itzg 차트 values 확장점: [차트 values.yaml](https://github.com/itzg/minecraft-server-charts/blob/master/charts/minecraft/values.yaml)
- 차트 버전 `5.1.2` 가용 (현재 `5.0.0`) — 도구 작업 시 bump 고려
- itzg `mc-health` probe `timeout=1s` 빡빡 — 도구 작업 시 timeout 상향 권장

## 알려진 함정

- **월드 포맷 업그레이드는 단방향**: 상위 버전 서버가 월드를 로드하면 `level.dat`이 신포맷으로 변환되어 하위 버전이 못 읽는다. 버전 다운그레이드 시 월드 데이터 처리 필요 (과거 26.1.2 → 1.21.11 다운그레이드에서 빈 월드 와이프로 해결).
- stateful 워크로드는 manifest가 git SSOT여도 **데이터 레이어는 git으로 롤백되지 않는다** — 버전 변경 전 데이터 영향을 먼저 검토할 것.
