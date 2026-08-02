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

- **OpenAI Codex OAuth** (ChatGPT 구독): `hermes auth add openai-codex --type oauth` = **device-code
  flow**(URL+코드, localhost 콜백 없음 → 헤드리스 OK). 크레덴셜은 **파일** `/opt/data/auth.json`의
  `credential_pool.openai-codex[]`에 저장 (env var 불가). billing = `subscription_included`
  → ChatGPT Plus/Pro 포함 사용량, **추가 과금 0**. `config.yaml` provider: openai-codex / default: gpt-5.3-codex.
- **seed-if-no-codex**: `seed` initContainer는 라이브 `config.yaml`/`auth.json`이 아직 `openai-codex`를
  참조하지 않을 때만 시드로 복사. Hermes가 소유(+토큰 in-place refresh)하면 grep 매칭 → 건너뜀 →
  refresh된 토큰·마이그레이션 config 미덮음.
- **SealedSecret `hermes-secrets`** (ns hermes): `HERMES_ADMIN_USERNAME`, `HERMES_ADMIN_PASSWORD`,
  `auth.json`(Codex 크레덴셜 파일), `API_SERVER_KEY`, `HERMES_DASHBOARD_BASIC_AUTH_SECRET`.
  (`TELEGRAM_BOT_TOKEN`은 후속 — env `optional: true` 배선.)

> ℹ️ Codex OAuth 토큰은 auth.json에서 자동 refresh. 4xx terminal 에러 시 refresh 토큰 dead 처리 →
> `hermes auth add openai-codex` 재발급 후 auth.json 재봉인.
> **Claude Max OAuth는 폐기** — 제3자 앱에 유료 extra-usage 크레딧만 소모(base 할당 미개방)라 사실상 종량제.
> **Gemini 무료티어**가 유일 $0 폴백 (`GOOGLE_API_KEY` env, provider gemini, Flash — 쿼터 제약).

---

## Phase B — Codex 크레덴셜 발급 (사람 작업, device-code)

헤드리스 파드는 OAuth 불가. 로컬 격리 컨테이너서 발급 → auth.json 생산:
```bash
mkdir -p ~/hermes-codex
docker run --rm -it --user $(id -u):$(id -g) \
  -e HERMES_HOME=/opt/data -e HOME=/opt/data \
  -v ~/hermes-codex:/opt/data --entrypoint hermes \
  nousresearch/hermes-agent:v2026.6.5 auth add openai-codex --type oauth
# URL+코드 → 브라우저서 ChatGPT 로그인/승인 → ~/hermes-codex/auth.json 생성
```

### 산출물 (Phase A 입력)
- `~/hermes-codex/auth.json` (`credential_pool.openai-codex[]`, access+refresh 토큰)
- admin user/pass (생성 위임 가능)
- (후속) Telegram 봇 토큰 — 유출 이력 있으니 revoke 후 신규

---

## Phase A — 봉인 + 배포

cert는 `k8s/sealed-secrets/cert.pem`에 커밋되어 있어 VPN/클러스터 접근 없이 sealing 가능.

**auth.json 봉인**: `cat ~/hermes-codex/auth.json | KUBECONFIG=~/.kube/config-json kubeseal --raw
--cert k8s/sealed-secrets/cert.pem --name hermes-secrets --namespace hermes --scope strict`
→ `sealed-secret.yaml`의 `auth.json` 키에 넣음. admin/API_SERVER_KEY/DASHBOARD_SECRET은 봉인 완료.
→ PR 머지하면 ArgoCD가 `apps/hermes.yaml` 자동 sync.

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
- Codex(ChatGPT 구독)로 추론 동작 (subscription_included, 종량제 미사용)
- `hermes.json-server.win` → Authentik 통과 후 대시보드 (admin: hermes / 봉인된 pass)
- (후속) Telegram 봇 ↔ 파드 대화 왕복

## 운영 메모

- 모델 변경: 대시보드 또는 파드 내 `hermes config set model.default <id>`. `config.yaml`(ConfigMap)은
  seed 전용 — 라이브 파일을 덮지 않음.
- 이미지 핀: 제3자 공개 이미지 → 수동 태그 업데이트 ([manifests/deployment.yaml](manifests/deployment.yaml)
  의 `v2026.6.5`). Image Updater 미사용.
- 백업: `/opt/data` = 기억/스킬/세션/인증 전부. PVC 유실 시 seed에서 재구성하되 OAuth는 재로그인 필요.
