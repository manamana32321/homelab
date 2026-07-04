# Hermes Agent

개인 상시 AI 비서 ([NousResearch/hermes-agent](https://github.com/NousResearch/hermes-agent), MIT).
영구기억 + 자율 스킬 + 메신저 게이트웨이 + credential pool. Claude **Max 구독 OAuth**로 추론
(종량제 API 키 미사용).

- 대시보드: https://hermes.json-server.win (Authentik forward-auth 게이트)
- 이미지: `nousresearch/hermes-agent:v2026.6.5` (Docker Hub, multiarch)
- 데이터: PVC `hermes-data` (longhorn-ssd, 5Gi) → `/opt/data` (`HERMES_HOME`/`HOME`)

## 구조

파드 1개 / 컨테이너 2개가 PVC `/opt/data`를 공유 (RWO → 단일 노드, `strategy: Recreate`):

| 컨테이너 | 명령 | 포트 | 노출 |
|---|---|---|---|
| `gateway` | `gateway run` | 8642 (OpenAI 호환 API) | ClusterIP 내부 전용 |
| `dashboard` | `dashboard --host 0.0.0.0 --port 9119 --no-open` | 9119 | Ingress (Authentik) |

Telegram은 **롱폴링 아웃바운드** → ingress 불필요. `terminal.backend: local` → 에이전트 셸이
파드 내부에서 실행 (파드 = 샌드박스, kubeconfig 미주입).

## 인증 / 시크릿

- **Claude Max OAuth**: `hermes model` → Anthropic → OAuth는 `claude setup-token`을 돌려
  장수명 토큰 `sk-ant-oat01-...`(~1년)을 발급, `.env`의 `ANTHROPIC_TOKEN`에 저장. 파드에는
  이 값을 **env var `ANTHROPIC_TOKEN`으로 주입** (`config.yaml` provider: anthropic + 이 토큰 +
  `ANTHROPIC_API_KEY` 미설정 → 구독 OAuth, 종량제 아님). 정적 장수명이라 refresh-rotation 무관.
- **seed-if-absent**: `seed` initContainer는 첫 부팅에만 `config.yaml`을 `/opt/data`로 복사.
  재시작 시 존재하면 건너뜀 → Hermes가 마이그레이션한 라이브 config를 stale seed가 덮지 않음.
  (크레덴셜은 파일 seed 아님 — env var.)
- **SealedSecret `hermes-secrets`** (ns hermes): `HERMES_ADMIN_USERNAME`, `HERMES_ADMIN_PASSWORD`,
  `ANTHROPIC_TOKEN`. (`TELEGRAM_BOT_TOKEN`은 후속 PR — env `optional: true`로 배선됨.)

> ℹ️ setup-token은 정적 장수명(~1년)이라 rotation 충돌 없음. 만료 시 `hermes model` 재발급 →
> `ANTHROPIC_TOKEN` 재봉인. 기존 `~/.claude/.credentials.json`(Claude Code 세션 토큰)은
> **재사용 금지** — scope·rotation 충돌 + 무관 MCP 토큰 다수 포함.

---

## Phase B — 데스크톱에서 크레덴셜 생산 (사람 작업)

헤드리스 파드는 브라우저 OAuth 불가. 데스크톱(또는 WSL `docker run`)에서 토큰을 받아 주입.

### Claude Max 토큰 (격리 컨테이너 — 호스트 `~/.claude` 안 건드림)
```bash
mkdir -p ~/hermes-data
docker run --rm -it -v ~/hermes-data:/opt/data \
  -e HERMES_UID=$(id -u) -e HERMES_GID=$(id -g) \
  nousresearch/hermes-agent:v2026.6.5 model
# → 1. Claude Pro/Max subscription (OAuth login) 선택
#    실제로는 `claude setup-token`을 돌려 sk-ant-oat 토큰 붙여넣기 흐름.
# 결과: ~/hermes-data/.env 의 ANTHROPIC_TOKEN (장수명 oat, ~1년)
```

### 산출물 (Phase A 입력)
- `~/hermes-data/.env` 의 `ANTHROPIC_TOKEN` (sk-ant-oat, 구독 OAuth — 종량제 아님)
- admin user/pass (직접 정하거나 생성 위임)
- (후속) Telegram 봇 토큰 — [reference_telegram_bot_reuse]는 유출 이력 있으니 revoke 후 신규

---

## Phase A — 봉인 + 배포

cert는 `k8s/sealed-secrets/cert.pem`에 커밋되어 있어 VPN/클러스터 접근 없이 sealing 가능.

**코어 3키 봉인 완료** (채워짐): `HERMES_ADMIN_USERNAME`, `HERMES_ADMIN_PASSWORD`, `ANTHROPIC_TOKEN`.
→ PR 머지하면 ArgoCD가 `apps/hermes.yaml` 자동 sync (Telegram 없이도 파드 Running).

Telegram 추가(후속 PR):
```bash
seal() { KUBECONFIG=~/.kube/config-json kubeseal --raw --cert k8s/sealed-secrets/cert.pem \
  --name hermes-secrets --namespace hermes --scope strict; }
echo -n "$TELEGRAM_BOT_TOKEN" | seal   # → sealed-secret.yaml encryptedData.TELEGRAM_BOT_TOKEN 추가
kubectl -n hermes rollout restart deploy/hermes   # optional env가 키를 집음
```

## 검증 (post-merge, hard-refresh 먼저)

```bash
kubectl -n argocd patch app hermes --type=merge \
  -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}'
argocd app wait hermes --sync --health --timeout=600
# SealedSecret 복호화 확인 (empty 함정):
kubectl -n hermes get secret hermes-secrets -o json | jq '.data | map_values(@base64d | length)'
kubectl -n hermes logs deploy/hermes -c gateway | grep -i "anthropic\|telegram\|provider"
```
- Claude Max로 추론 동작 (종량제 API 키 미사용)
- `hermes.json-server.win` → Authentik 통과 후 대시보드 (admin: hermes / 봉인된 pass)
- (후속) Telegram 봇 ↔ 파드 대화 왕복

## 운영 메모

- 모델 변경: 대시보드 또는 파드 내 `hermes config set model.default <id>`. `config.yaml`(ConfigMap)은
  seed 전용 — 라이브 파일을 덮지 않음.
- 이미지 핀: 제3자 공개 이미지 → 수동 태그 업데이트 ([manifests/deployment.yaml](manifests/deployment.yaml)
  의 `v2026.6.5`). Image Updater 미사용.
- 백업: `/opt/data` = 기억/스킬/세션/인증 전부. PVC 유실 시 seed에서 재구성하되 OAuth는 재로그인 필요.
