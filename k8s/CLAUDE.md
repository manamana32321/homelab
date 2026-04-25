# k8s/ — Claude Code Guidance

> homelab K3s 매니페스트 작성 시 반드시 따라야 할 컨벤션.

## Secret 관리 — 도메인 바운드 SSOT

### 핵심 원칙

> **"Secret을 발행하는 도메인의 ns가 owner. Consumer는 Reflector로 받는다."**

발행자(issuer)와 사용자(consumer)를 혼동하지 말 것.

| Bad ❌ | Good ✅ |
|---|---|
| `consumer-app/secrets`에 모든 의존 키를 넣고 sealing | 각 키를 발행자 ns가 보관, consumer는 reflect로 받음 |
| 같은 키를 여러 SealedSecret에 중복 sealing | 단일 SOT, rotation 시 한 곳만 변경 |
| 모든 키를 `shared-secrets` ns에 모아둠 | 도메인 owner가 명확하면 그 ns. 진짜 cross-domain만 별도 ns |

### Owner 판단 기준

1. **이 secret을 누가 발행/생성하는가?** → 그 ns가 owner
   - `health-hub-api-token` → health-hub backend가 발행 → `health-hub` ns
   - `alertmanager-telegram` → 알림 시스템 owns → `observability` ns
2. 외부 서비스 토큰 (Canvas, Google OAuth 등) → 그 외부 서비스를 wrap하는 cluster 워크로드 ns
   - 예: Canvas 토큰 → essentia ns (essentia-api가 LearningX wrap)
3. 진짜 cross-domain (LLM 키 등) → consumer hub ns or 신규 `shared-secrets`
4. 앱 자체가 발행하는 secret (예: webhook secret) → 자기 ns

### Internal vs Public split

발행자 ns의 secret이 여러 키를 가질 때:
- **Internal 전용** (DB password, MCP token 등): 그대로 `<app>-secret`에 유지, **reflect 안 함**
- **외부 노출 가능** (public API token 등): 별도 SealedSecret로 split → Reflector annotation

예: `health-hub-secret` → 두 개로:
- `health-hub-secret` (DB_PASSWORD, MCP_AUTH_TOKEN — internal)
- `health-hub-api-token` (API_TOKEN — reflectable)

### Reflector 사용

Consumer ns로 자동 복제:
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

복수 ns: `"automation,morton-prod"`로 콤마 구분.

### Consumer 측 사용

Deployment / CronJob env에서:
```yaml
- name: HEALTH_HUB_TOKEN
  valueFrom:
    secretKeyRef:
      name: health-hub-api-token  # ← reflected from health-hub ns
      key: API_TOKEN
```

같은 ns에 자동 복제됐으니 그냥 참조하면 됨.

### Anti-patterns

- ❌ 한 SealedSecret에 5+ 다른 도메인 키 (CANVAS + TELEGRAM + HEALTH_HUB ...)
- ❌ Consumer ns에서 자기 도메인 아닌 키를 새로 sealing
- ❌ Reflector annotation 누락한 채 cross-ns에서 secret 만들기 (drift 생김)
- ❌ DB password 같은 internal secret을 reflectable로 노출

### SealedSecret 생성 명령

```bash
# cert.pem 위치 확인 (k8s/sealed-secrets/cert.pem)
echo -n "$VALUE" | KUBECONFIG=~/.kube/config-json kubeseal --raw \
  --cert k8s/sealed-secrets/cert.pem \
  --name <secret-name> \
  --namespace <ns> \
  --scope strict
```

cert는 30일마다 로테이션. 만료 시 VPN 연결 후 `kubeseal --fetch-cert` 재발급.

---

## DNS 레코드 관리 — 항상 Terraform

### 핵심 원칙

> **모든 DNS 레코드는 `cloudflare/dns.tf`에 등록한다. Cloudflare 웹 UI 직접 추가 금지.**

이유: drift 방지, 코드 리뷰 가능, 롤백 가능, 다른 환경 복제 용이.

### 레코드 추가 패턴

```hcl
resource "cloudflare_record" "<unique_name>" {
  zone_id = cloudflare_zone.main.id
  name    = "<subdomain>"  # e.g., "brain-agent"
  type    = "A"
  content = var.default_ip
  proxied = false  # 또는 true (아래 가이드)
  comment = "<짧은 설명>"
}
```

### `proxied` 결정 가이드

| 케이스 | proxied | 이유 |
|---|---|---|
| **cert-manager DNS-01 검증** | `false` | Cloudflare proxy 거치면 인증서 발급 영향 가능 |
| **Multi-level subdomain** (예: `api.amang`) | `false` | Cloudflare 무료 Universal SSL이 1단계만 커버. cert-manager로 발급 |
| **단일 subdomain + proxy 이점 활용** | `true` | DDoS 보호, TLS termination, edge caching |
| **Webhook receiver** | `false` 권장 | Cloudflare가 timeout 짧음 (응답 지연 시 retry 폭증) |
| **Vercel CNAME** | `false` | Vercel 자체 SSL |

기존 레코드 그루핑 참고:
- `proxied = true`: argocd, auth, db, prometheus, grafana, otel, habits, health, essentia, ...
- `proxied = false`: brain-agent, k8s, longhorn, frigate, ha, photos, files, mc, *.amang ...

### 서브도메인 컨벤션

`(service).project.(env).json-server.win`

| 규칙 | 예시 |
|---|---|
| 독립 서비스 | `grafana.json-server.win` |
| 프로젝트 web (prod) | `amang.json-server.win` |
| 프로젝트 web (staging) | `amang.staging.json-server.win` |
| 프로젝트 service (prod) | `api.amang.json-server.win` |
| 프로젝트 service (staging) | `api.amang.staging.json-server.win` |

### Wildcard 사용

다중 서비스 묶음은 wildcard로:
```hcl
resource "cloudflare_record" "wildcard_amang" {
  name    = "*.amang"
  ...
  proxied = false  # multi-level, cert-manager TLS
}
```

### 적용 절차

```bash
cd cloudflare
terraform plan   # 변경 확인 (반드시)
terraform apply  # 사용자 승인 후
```

**`-auto-approve` 절대 금지** — root rule.

### Email 라우팅 주의

`mx_route1/2/3`, `dkim_cf2024`는 **Cloudflare Email Routing이 자동 관리** — Terraform에서 제외.

---

## 그 외 공통 패턴 (참고)

### ArgoCD Application 위치

```
k8s/argocd/applications/
├── infra/         # cert-manager, external-secrets, reflector, sealed-secrets, ingress 등
├── observability/ # prometheus, grafana, loki, tempo, alertmanager
└── apps/          # 실제 워크로드
```

신규 Application은 카테고리 맞춰 배치. `apps` app-of-apps가 자동 sync.

### ConfigMap에 Python script 임베딩

`configmap.yaml`에 Python을 직접 embed하면 YAML parser가 `{...}`, `[...]`를 flow-map으로 오인. 깔끔한 방법:

```yaml
# kustomization.yaml
configMapGenerator:
  - name: <app>-script
    files:
      - script.py

generatorOptions:
  disableNameSuffixHash: true  # consumer가 ConfigMap 이름 안정적으로 참조
```

`script.py` 별도 파일 → IDE syntax highlighting + git diff 명료.

### 스토리지 클래스

| StorageClass | 용도 |
|---|---|
| `longhorn-ssd` (xfs) | DB·stateful 앱, Prometheus |
| `local-path-hdd-samsung` | 텔레메트리 (Loki, Tempo, Grafana) |
| hostPath `/mnt/hdd-seagate-1t` | 미디어 (Immich, Seafile, Frigate) |

raspi-1에 PV 배치 최후 (SD카드 IO 느림).

### main 커밋 금지

- main 직접 커밋 ❌
- 워크트리 → feat 브랜치 → PR → 승인 후 머지
- 긴급 핫픽스만 예외 (서비스 장애 등)
