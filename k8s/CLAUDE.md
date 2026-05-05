# k8s/ — Claude Code Guidance

> homelab K3s 매니페스트 작성 시 반드시 따라야 할 컨벤션.
> 이 파일이 k8s 작업의 **선언적 SSOT**. 로컬 설정 파일에 쪼개지 말 것.

## 최상위 원칙

- 모든 K8s 리소스는 **ArgoCD로 관리** — 수동 `kubectl apply` 금지 (디버깅·긴급 시 제외)
- **멀티환경 분리**: Kustomize `base/` + `overlays/[env]/`, ApplicationSet 사용
- **main 커밋 금지** — worktree → feat 브랜치 → PR → 승인 후 머지
- 긴급 핫픽스는 예외 (서비스 장애 시 main 직접 허용)

---

## Secret 관리 — 도메인 바운드 SSOT

### 핵심 원칙

> **"Secret을 발행하는 도메인의 ns가 owner. Consumer는 Reflector로 받는다."**

| Bad ❌ | Good ✅ |
|---|---|
| Consumer ns에 모든 의존 키를 넣고 sealing | 각 키를 발행자 ns가 보관, consumer는 reflect |
| 같은 키를 여러 SealedSecret에 중복 | 단일 SOT, rotation 시 한 곳만 변경 |
| 모든 키를 `shared-secrets` ns에 몰기 | 도메인 owner가 명확하면 그 ns |

### Owner 판단 기준

1. **누가 발행/생성하는가?** → 그 ns가 owner
   - `health-hub-api-token` → health-hub backend → `health-hub` ns
   - `alertmanager-telegram` → 알림 시스템 → `observability` ns
2. **외부 서비스 토큰** (Canvas, Google OAuth 등) → 그 외부 서비스를 wrap하는 cluster 워크로드 ns
   - 예: Canvas 토큰 → essentia ns
3. **진짜 cross-domain** (LLM 키 등) → consumer hub ns (예: `automation`)
4. **앱 자체가 발행**하는 secret → 자기 ns

### Internal vs Public split

발행자 ns의 secret이 여러 키를 가질 때:
- **Internal 전용** (DB password, internal token 등): 그대로 유지, reflect 안 함
- **외부 노출 가능** (public API token): 별도 SealedSecret + Reflector annotation

예: `health-hub-secret` 분리
- `health-hub-secret` (DB_PASSWORD, MCP_AUTH_TOKEN — internal)
- `health-hub-api-token` (API_TOKEN — reflectable)

### Reflector 사용

```yaml
spec:
  template:
    metadata:
      annotations:
        reflector.v1.k8s.emberstack.com/reflection-allowed: "true"
        reflector.v1.k8s.emberstack.com/reflection-allowed-namespaces: "automation"
        reflector.v1.k8s.emberstack.com/reflection-auto-enabled: "true"
        reflector.v1.k8s.emberstack.com/reflection-auto-namespaces: "automation"
```

복수 ns: `"automation,morton-prod"` 콤마 구분.

### Consumer 측 사용

Deployment / CronJob env에서 reflected secret 그대로 참조:
```yaml
- name: HEALTH_HUB_TOKEN
  valueFrom:
    secretKeyRef:
      name: health-hub-api-token  # ← reflected from health-hub ns
      key: API_TOKEN
```

### Anti-patterns

- ❌ 한 SealedSecret에 5+ 다른 도메인 키
- ❌ Consumer ns에서 자기 도메인 아닌 키 sealing
- ❌ Reflector annotation 누락 채 cross-ns secret 만들기
- ❌ DB password 같은 internal을 reflectable로 노출

### sealing 명령 컨벤션

```bash
# 타겟 클러스터 명시 필수 (global hook으로 강제됨)
echo -n "$VALUE" | KUBECONFIG=~/.kube/config-json kubeseal --raw \
  --cert k8s/sealed-secrets/cert.pem \
  --name <secret-name> \
  --namespace <ns> \
  --scope strict
```

cert는 30일마다 로테이션. 만료 시 VPN 연결 + `kubeseal --fetch-cert` 재발급.

---

## DNS 레코드 관리 — 항상 Terraform

> 참조 파일: [`cloudflare/dns.tf`](../cloudflare/dns.tf) (homelab repo 루트 기준).
> 기존 레코드 패턴·컨벤션은 이 파일에서 읽을 것.

### 핵심 원칙

> **모든 DNS 레코드는 `cloudflare/dns.tf`에 등록. Cloudflare 웹 UI 직접 추가 금지.**

### 레코드 추가 패턴

```hcl
resource "cloudflare_record" "<unique_name>" {
  zone_id = cloudflare_zone.main.id
  name    = "<subdomain>"
  type    = "A"
  content = var.default_ip
  proxied = false  # 아래 가이드
  comment = "<짧은 설명>"
}
```

### `proxied` 결정 가이드

| 케이스 | proxied | 이유 |
|---|---|---|
| cert-manager DNS-01 검증 | `false` | Cloudflare proxy 영향 회피 |
| Multi-level subdomain (`api.amang`) | `false` | Universal SSL 1단계만 커버, cert-manager 사용 |
| 단일 subdomain + proxy 이점 활용 | `true` | DDoS·TLS·edge cache |
| Webhook receiver | `false` | Cloudflare timeout 회피 |
| Vercel CNAME | `false` | Vercel 자체 SSL |

### 서브도메인 컨벤션

`(service).(project).(env).json-server.win`

| 규칙 | 예시 |
|---|---|
| 독립 서비스 | `grafana.json-server.win` |
| 프로젝트 web (prod) | `amang.json-server.win` |
| 프로젝트 service (prod) | `api.amang.json-server.win` |
| 프로젝트 service (staging) | `api.amang.staging.json-server.win` |

### 적용 절차

```bash
cd cloudflare
terraform plan   # 반드시 확인
terraform apply  # 승인 프롬프트에서 yes
```

`-auto-approve` 절대 금지.

### Email 라우팅 주의

`mx_route1/2/3`, `dkim_cf2024`는 Cloudflare Email Routing 자동 관리 — Terraform 제외.

---

## ConfigMap 관리

### 코드·스크립트 임베딩 vs 별도 파일

- **별도 파일 + `configMapGenerator`** (권장)
  - 5줄+ 코드 (Python, shell, JSON)
  - YAML parser flow-map 오인 회피
  - IDE syntax highlighting + git diff 명료
- **inline `data:`**:
  - 단순 key-value, < 5줄
  - kustomization과 같이 봐야 하는 구성

```yaml
# kustomization.yaml
configMapGenerator:
  - name: <app>-script
    files:
      - script.py

generatorOptions:
  disableNameSuffixHash: true  # consumer가 stable name 참조
```

### 핵심 옵션

- `disableNameSuffixHash: true`: CronJob/Deployment의 `configMap.name` 안정 참조. 변경 시 **수동 pod restart 필요**
  ```bash
  kubectl rollout restart deploy/<name> -n <ns>
  ```
- `disableNameSuffixHash: false` (default): hash suffix → 변경 시 자동 rolling

### 마운트 vs env 주입

| 패턴 | 용도 |
|---|---|
| Volume mount | 앱이 파일 read (Python script, config.yaml) |
| `envFrom` | 환경변수 배치 주입 (모든 키 다 필요) |
| `env.valueFrom.configMapKeyRef` | 개별 키만 |

### Anti-patterns

- ❌ Sensitive 데이터 ConfigMap에 (SealedSecret으로)
- ❌ generic 이름 (`config`, `settings`) → `<app>-<purpose>`
- ❌ inline에 50+줄 코드 embed

---

## Health Probes

### 3종류 역할

| 프로브 | 실패 시 | 용도 |
|---|---|---|
| `livenessProbe` | pod **kill + restart** | "프로세스가 죽었나" |
| `readinessProbe` | service endpoint에서 **제거** | "트래픽 받을 준비 됐나" |
| `startupProbe` | slow-start 보호 (있으면 liveness/readiness 지연) | DB 마이그레이션 등 |

### 커스텀 앱 구현 원칙

- **HTTP `GET /healthz`** 표준 — self-check only (DB·외부 API 호출 ❌)
- liveness + readiness 같은 endpoint, **timing 다름**
- 외부 오픈소스 앱: 기존 endpoint 활용. 없으면 **무리하게 추가 X**

### 권장 timing

```yaml
readinessProbe:
  httpGet: { path: /healthz, port: http }
  initialDelaySeconds: 10   # 빨리 시작 신호
  periodSeconds: 5          # 자주 체크
  failureThreshold: 3       # 15초 후 endpoint에서 빠짐

livenessProbe:
  httpGet: { path: /healthz, port: http }
  initialDelaySeconds: 60   # false positive 방지
  periodSeconds: 30         # 덜 자주
  failureThreshold: 3       # 90초 실패 후 kill
```

### tcpSocket / exec 사용

- **tcpSocket**: DB·메시지 큐·gRPC 등 HTTP 아닌 서비스
  - 예: health-hub의 `mcp` container (port 8081)
- **exec**: 커스텀 shell 체크 (드물게)

### Anti-patterns

- ❌ liveness에서 DB 쿼리 → DB 장애 시 restart 폭증
- ❌ liveness에서 외부 API 호출 (OpenAI 등) → 외부 의존 사고 전파
- ❌ `failureThreshold: 1` → false positive restart loop
- ❌ readiness만 있고 liveness 없음 → 죽은 pod 영원히 남음
- ❌ healthz가 무조건 200 반환 → hang 감지 불가

---

## 외부·내부 프로빙 (Blackbox Exporter)

### 2-레이어 관측

1. **내부** (cluster 안에서 ClusterIP 체크)
   - `blackbox-exporter` Deployment (observability ns)
   - 등록: `k8s/observability/blackbox-exporter/values.yaml`의 `targets` 배열
   - 현재 대상: argocd / grafana / prometheus (cluster.local URL)
   - 목적: 서비스 self-health

2. **외부** (인터넷 → ingress → cluster, 독립 관측점)
   - **Coral Dev Board**의 Blackbox Exporter (cluster 밖)
   - 대상: **`argocd.json-server.win` 단 하나** (대표 endpoint)
   - 목적: DNS·Cloudflare·ingress·TLS 전체 경로 확인
   - Prometheus 설정: `additionalScrapeConfigs`의 `coral-blackbox-http`

### 새 서비스 배포 시

- **내부 targets에 추가** (service health 기본)
- 외부 probing은 **argocd 하나 유지** (대표성, 경로 공유)

### 알림

`k8s/observability/prometheus/manifests/alert-rules.yaml`의 `blackbox` 그룹:
- `EndpointDown` (probe_success == 0, 2분+)
- `TLSCertExpiringSoon` (< 14일)
- `SlowResponse` (avg > 3초, 5분+)
- 모두 `source!="coral"` 필터로 중복 방지

---

## Resources requests/limits

### 원칙

- **requests**: 스케줄러 배치 하한 — **반드시 설정**
- **limits**: OOM kill 임계치 — 보통 requests의 2~4배
- `limits` 없으면 노드 OOM 시 전체 터질 위험

### 기본 템플릿 (커스텀 앱)

```yaml
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 256Mi
```

### 특수 케이스

- **Python + pip install 인라인** (brain-agent 등): `memory: 256Mi-512Mi` 권장
- **Postgres·TimescaleDB**: `request 256Mi / limit 1Gi+`
- **Prometheus**: metric 볼륨에 따라 `500Mi+`
- **minio-operator**: CPU 200m→500m 상향 필요 (PolicyBinding watch CPU 소모)

---

## Labels 표준

Kubernetes 공식 label (디버깅·쿼리 일관성):

```yaml
labels:
  app.kubernetes.io/name: brain-agent-webhook     # 앱 이름
  app.kubernetes.io/part-of: automation           # 상위 그룹
  app.kubernetes.io/component: webhook            # 역할
  app.kubernetes.io/managed-by: argocd            # 관리 도구 (보통 argocd가 자동)
```

최소 `app.kubernetes.io/part-of`만이라도 붙이면 label selector로 묶음 쿼리 가능:
```bash
kubectl get all -A -l app.kubernetes.io/part-of=automation
```

---

## Image Updater

### 사용 의무

- **커스텀 앱 이미지 = ArgoCD Image Updater 필수**
- `latest` 태그 + 수동 재시작 **금지**
- 공개 이미지 (nginx, postgres 등)는 Image Updater 불필요 (Helm chart의 `image.tag`로 pin)

### 전략

- **`digest`** 권장 — immutable pin
- **`semver`** 필요 시 (Helm chart release tracking)

### Write-back mode: **`argocd` 모드 사용** (git 모드 비선호)

annotation: `argocd-image-updater.argoproj.io/write-back-method: argocd`

| 모드 | 동작 | 채택? |
|---|---|---|
| **argocd** (현 사용) | digest override를 ArgoCD Application CR의 `spec.source.helm.parameters` 또는 `spec.source.kustomize.images`에 직접 박음. **git push 안 함** | ✅ 모든 앱 |
| git | repo의 `kustomization.yaml`에 commit/push | ❌ 사용 안 함 |

**왜 argocd 모드인가**:
- Image Updater가 repo PAT token 갱신할 일 없음 → silent failure 한 종류 제거
- ArgoCD API 호출만 하면 끝. 빠르고 단순
- Trade-off: digest 변경 이력이 git log 아닌 etcd에만 남음 (audit trail 약함). 홈랩 환경에선 수용 가능

### 패턴

1. 앱 소스 repo GitHub Actions: 이미지 빌드 → GHCR push (digest tag)
2. Image Updater가 GHCR 폴링 (default 2m) → 새 digest 감지
3. ArgoCD API 호출로 Application CR 수정 (override 추가)
4. ArgoCD가 spec 변경 감지 → sync → rollout

**ArgoCD Application annotation 예시** (essentia):
```yaml
metadata:
  annotations:
    argocd-image-updater.argoproj.io/image-list: "api=ghcr.io/essentia-edu/api:latest, web=ghcr.io/essentia-edu/web:latest"
    argocd-image-updater.argoproj.io/api.update-strategy: digest
    argocd-image-updater.argoproj.io/api.pull-secret: "pullsecret:essentia/ghcr-essentia"
    argocd-image-updater.argoproj.io/write-back-method: argocd  # ← 명시 필수
```

**관련 이슈**: Image Updater가 private GHCR pullSecret을 읽으려면 RBAC 필요 — [homelab#139](https://github.com/manamana32321/homelab/pull/139)에서 해결됨.

---

## Storage 노드 배치

### StorageClass 매핑

| StorageClass | 노드·디스크 | 용도 |
|---|---|---|
| `longhorn-ssd` (xfs) | server-1 + server-2 SSD | DB·stateful 앱, Prometheus |
| `local-path-hdd-samsung` | server-1 Samsung HDD | 텔레메트리 (Loki, Tempo, Grafana) |
| hostPath `/mnt/hdd-seagate-1t` | server-1 Seagate 1TB | 미디어 (Immich, Seafile, Frigate) |

### nodeSelector 규칙

**raspi-1엔 PV 배치 최후순위** (SD카드, IO 느림). 필요 시 명시적으로 제외:
```yaml
nodeSelector:
  kubernetes.io/hostname: json-server-1
```

**hostPath PV + 특정 노드**: 반드시 `nodeSelector`로 고정
- Frigate: json-server-1 (GPU)
- 미디어 hostPath: json-server-1 (물리적 디스크 위치)

### 노드 특이사항

- **json-server-1**: 4C/8T, 16GB, GTX 750 Ti, SSD 240GB + HDD 2개 (Samsung·Seagate)
  - Samsung HDD 건강 주의 (Pending_Sector 2, 2026-12 교체 예정)
- **json-server-2**: 6C/6T, 16GB, SSD 119GB. GPU 없음
- **raspi-1**: RPi 4B 8GB, SD카드. PV 배치 최후

---

## DB Migration

### 원칙

**Job으로 실행** (NOT initContainer).

| 방식 | 실행 빈도 | 문제 |
|---|---|---|
| initContainer | pod 시작마다 | idempotent 필수, 느림, rolling restart마다 실행 |
| **Job** | one-shot | 명확, 로그 분리, 실패 명시적 |

### 구현 패턴

**ArgoCD sync hook + wave**:
```yaml
metadata:
  annotations:
    argocd.argoproj.io/hook: PreSync
    argocd.argoproj.io/hook-delete-policy: BeforeHookCreation
    argocd.argoproj.io/sync-wave: "-1"  # 앱 배포 (wave 0) 전에 실행
```

### 체크

- Job의 `backoffLimit` 낮게 (기본 6 → 2-3)
- `activeDeadlineSeconds`로 최대 실행 시간 제한
- 실패 시 Alertmanager 알림 (Prometheus `kube_job_failed`)

---

## ArgoCD Application 배치

```
k8s/argocd/applications/
├── infra/         # cert-manager, external-secrets, reflector, sealed-secrets, ingress 등
├── observability/ # prometheus, grafana, loki, tempo, alertmanager
└── apps/          # 실제 워크로드 (brain-agent, health-hub, essentia, ...)
```

신규 Application은 카테고리 맞춰 배치. `apps` app-of-apps가 자동 sync.

### Helm chart Application 패턴

Multi-source 사용 — chart + values ref 분리:
```yaml
sources:
  - repoURL: <helm-repo-url>
    chart: <chart-name>
    targetRevision: <version>
    helm:
      valueFiles:
        - $values/k8s/<app>/values.yaml
  - repoURL: https://github.com/manamana32321/homelab.git
    targetRevision: main
    ref: values
```

### Sync wave

마이그레이션·prerequisite 먼저 실행 시 `sync-wave` annotation 사용 (기본 0).

---

## 체크리스트 — 새 앱 배포 시

1. [ ] Secret은 발행자 ns에 owner로, consumer는 Reflector
2. [ ] DNS는 `cloudflare/dns.tf`에 추가 (필요 시)
3. [ ] Deployment에 healthz probe 2개 (liveness·readiness) + 타이밍 다르게
4. [ ] Resources requests/limits 둘 다 설정
5. [ ] Labels: 최소 `app.kubernetes.io/part-of`
6. [ ] 커스텀 이미지면 Image Updater 설정
7. [ ] Storage 필요 시 StorageClass 선택 + nodeSelector (hostPath면 필수)
8. [ ] DB 마이그레이션 있으면 Job + PreSync hook
9. [ ] Blackbox-exporter 내부 targets에 서비스 추가
10. [ ] ArgoCD Application 생성 (`apps/`·`infra/`·`observability/` 중 적절히)
