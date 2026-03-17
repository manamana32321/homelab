# 래미 에이전트 Fly.io 이전 설계

> GitHub Issue: [#59](https://github.com/manamana32321/homelab/issues/59)
> 작성일: 2026-03-17

## 배경

현재 래미(main agent)가 홈랩 K3s 클러스터 내 OpenClaw gateway에서 실행 중.
홈랩 클러스터에는 사생활 정보(사진, 파일 등)가 있어 AI 에이전트와 보안 격리가 필요.
래미를 외부 관리형 서비스(Fly.io)로 이전하여 홈랩과 완전 분리한다.

## 결정 사항

| 항목 | 결정 |
|------|------|
| 배포 대상 | Fly.io (관리형 컨테이너) |
| 접근 방식 | Blue-Green (외부 먼저 → 홈랩 정리) |
| 래미 채널 | Slack only (Telegram, Discord 사용 안 함) |
| 홈랩 연결 | 완전 차단 (래미는 홈랩 몰라야 함) |
| 홈랩 잔류 에이전트 | 자비스, 건강, 관계, 인터뷰, 오픈캠퍼스 (자비스가 default 승격) |
| ConfigMap 정리 | `openclaw-config`의 `openclaw.json` 비우기 (런타임 SSOT) |
| State 관리 | Fly Volume + 자동 스냅샷 (git 레포 백업 없음) |
| 설정 레포 | `mortonCareer/openclaw` (private) |

## 아키텍처

```
[Fly.io - nrt region]               [홈랩 K3s]
┌──────────────────────┐            ┌─────────────────────┐
│ OpenClaw (래미 전용)   │            │ OpenClaw (잔류)      │
│ shared-cpu-4x / 2GB  │            │ - 자비스 (default)   │
│ Fly Volume 10GB      │            │ - 건강, 관계         │
│                      │            │ - 인터뷰, 오픈캠퍼스  │
│ 채널: Slack only     │            └─────────────────────┘
│ 시크릿: fly secrets  │
│ 스냅샷: 일 1회/60일  │            ↕ 연결 없음
└──────────────────────┘
```

## Fly.io 구성

### 레포 구조

```
mortonCareer/openclaw (private)
├── fly.toml
├── .gitignore
└── README.md
```

### fly.toml

```toml
app = "remi-openclaw"
primary_region = "nrt"  # Tokyo (한국 최근접)

[build]
  image = "ghcr.io/openclaw/openclaw:latest"

[env]
  NODE_ENV = "production"
  NODE_OPTIONS = "--max-old-space-size=1536 --dns-result-order=ipv4first --no-network-family-autoselection"

[mounts]
  source = "openclaw_data"
  destination = "/home/node"

[[services]]
  internal_port = 18789
  protocol = "tcp"
  [[services.ports]]
    port = 443
    handlers = ["tls", "http"]

[processes]
  app = "node openclaw.mjs gateway --allow-unconfigured --bind lan"

[[vm]]
  size = "shared-cpu-4x"
  memory = "2gb"
```

### 비용

| 항목 | 월 비용 |
|------|---------|
| shared-cpu-4x, 2GB RAM | ~$13.27 |
| 10GB Fly Volume | $1.50 |
| 스냅샷 (10GB, 첫 10GB 무료) | $0 |
| **합계** | **~$15** |

### 스냅샷 설정

- 자동 스냅샷: 일 1회 (기본 활성)
- 보관 기간: 60일 (`--snapshot-retention 60`)
- 수동 스냅샷: `fly volumes snapshots create <vol-id>` (필요 시)

## 시크릿 배분

| ID | 키 | 래미 (Fly.io) | 홈랩 (K3s) |
|---|---|---|---|
| 1 | `ANTHROPIC_SETUP_TOKEN` | ✅ 동일값 공유 | ✅ 유지 |
| 2 | `TELEGRAM_BOT_TOKEN` (default) | ❌ | ❌ 제거 |
| 3 | `SLACK_BOT_TOKEN` + `SLACK_APP_TOKEN` | ✅ 이전 | ❌ 제거 |
| 4 | `DISCORD_BOT_TOKEN` | ❌ | ❌ 제거 |
| 5 | `OPENAI_API_KEY` | ✅ 회사 키 | ✅ 개인 키 (별도) |
| 6 | `TELEGRAM_BOT_TOKEN_JARVIS` | ❌ | ✅ 유지 |
| 7 | `TELEGRAM_BOT_TOKEN_STUDY` | ❌ | ✅ 유지 |
| 8 | health/relationship/interview 토큰 | ❌ | ✅ 유지 (런타임) |
| 9 | `CANVAS_ACCESS_TOKEN` | ❌ | ✅ 유지 |
| 10 | `OPENCLAW_GATEWAY_TOKEN` | ✅ 새로 발급 | ✅ 기존 유지 |
| 11 | `NOTION_TOKEN` | ✅ 이전 | ❌ 제거 |
| 12 | `SEAFILE_TOKEN` / `SEAFILE_URL` | ❌ | ✅ 유지 (오픈캠퍼스용) |

## 전환 절차 (Blue-Green)

### Phase 1: Fly.io에 래미 배포

1. `mortonCareer/openclaw` private 레포 생성
2. `fly launch` → app 생성 (`remi-openclaw`, nrt region)
3. `fly volumes create openclaw_data --size 10 --region nrt --snapshot-retention 60`
4. `fly secrets set ANTHROPIC_SETUP_TOKEN=... SLACK_BOT_TOKEN=... SLACK_APP_TOKEN=... OPENAI_API_KEY=... OPENCLAW_GATEWAY_TOKEN=... NOTION_TOKEN=...`
5. `fly deploy`
6. gateway 웹UI 접속 확인

### Phase 2: Slack 전환

1. 홈랩 OpenClaw GUI에서 Slack 비활성화 (`enabled: false`)
2. Fly.io 래미의 Slack Socket Mode 연결 확인
3. Slack 메시지로 래미 응답 검증

### Phase 3: 홈랩 정리 (PR)

**homelab 레포 변경:**

1. `k8s/openclaw/manifests/configmap.yaml`
   - `openclaw-config`의 `openclaw.json` 데이터를 `{}` 로 비우기

2. `k8s/openclaw/manifests/sealed-secret.yaml`
   - 제거 키: `TELEGRAM_BOT_TOKEN`, `SLACK_BOT_TOKEN`, `SLACK_APP_TOKEN`, `DISCORD_BOT_TOKEN`
   - 잔류 키: `ANTHROPIC_SETUP_TOKEN`, `OPENAI_API_KEY`, `CANVAS_ACCESS_TOKEN`, `OPENCLAW_GATEWAY_TOKEN`, `TELEGRAM_BOT_TOKEN_JARVIS`, `TELEGRAM_BOT_TOKEN_STUDY`
   - SealedSecret 재생성 필요 (kubeseal)

3. 런타임 config 정리 (kubectl cp 또는 GUI)
   - `agents.list`에서 래미 제거, 자비스를 `default: true`로 승격
   - `channels.telegram.accounts.default` 제거
   - `channels.slack` → `enabled: false`
   - `channels.discord` 제거
   - `env.vars`에서 `NOTION_TOKEN` 제거
   - `bindings`에서 래미 관련 제거 (현재 런타임에 명시적 래미 바인딩 없으므로 확인만)

4. PR 머지 → ArgoCD 자동 sync

### Phase 4: 검증

1. 홈랩 OpenClaw 재시작 후 자비스 등 정상 동작 확인
2. Fly.io 래미 Slack 응답 안정성 확인
3. 래미에서 홈랩 내부 서비스 접근 불가 확인

## 홈랩 ConfigMap 현황 vs 변경

현재 `openclaw-config`의 `openclaw.json`은 런타임과 거의 전 항목이 겹치거나 충돌.
런타임 config (PVC)이 SSOT이므로 ConfigMap 데이터를 `{}`로 비운다.
`openclaw-env` ConfigMap (NODE_OPTIONS 등)은 유지 — 이건 K8s 환경 설정이라 런타임과 겹치지 않음.
