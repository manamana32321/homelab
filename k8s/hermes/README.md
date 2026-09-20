# Hermes Agent

개인 상시 AI 비서 ([NousResearch/hermes-agent](https://github.com/NousResearch/hermes-agent), MIT).
영구기억 + 자율 스킬 + 메신저 게이트웨이 + credential pool. **Codex(ChatGPT 구독 OAuth)**로 추론
(billing `subscription_included`, 종량제 API 키 미사용).

- 대시보드: https://hermes.json-server.win (Authentik forward-auth 게이트)
  - Authentik 쪽 provider/application/정책은 [blueprint-forward-auth.yaml](../authentik/manifests/blueprint-forward-auth.yaml)
    에 선언한다. Ingress 미들웨어만 붙이고 여기 등록을 빠뜨리면 Authentik 이 404 를 낸다.
  - 대시보드 계정은 SealedSecret `hermes-secrets` 의 `HERMES_ADMIN_USERNAME` / `HERMES_ADMIN_PASSWORD`
- 이미지: `nousresearch/hermes-agent:v2026.6.5` (Docker Hub, multiarch)
- 데이터: PVC `hermes-data` (longhorn-ssd, 5Gi) → `/opt/data` (`HERMES_HOME`/`HOME`)

## 구조

파드 1개 / 컨테이너 2개가 PVC `/opt/data`를 공유 (RWO → 단일 노드, `strategy: Recreate`):

| 컨테이너 | 명령 | 포트 | 노출 |
|---|---|---|---|
| `gateway` | `gateway run` | 8642 (OpenAI 호환 API) | ClusterIP 내부 전용 |
| `dashboard` | `dashboard --host 0.0.0.0 --port 9119 --no-open` | 9119 | Ingress (Authentik) |

Discord는 **아웃바운드 WebSocket(Discord gateway)** → ingress·포트개방 불필요.
`terminal.backend: local` → 에이전트 셸이 파드 내부에서 실행 (파드 = 샌드박스, kubeconfig 미주입).

## Discord 게이트웨이

서버 `1540220813477945444` (개인 홈랩 서버, Alertmanager 알람이 오는 곳).

| 채널 | ID | Hermes 동작 |
|---|---|---|
| `#일반` | `1540220813993840664` | 멘션 없이 대화(free-response), 스레드 없이 직접 답변, cron·자율 알림 발신(home) |
| `#alert` | `1540223647350915134` | 멘션 시에만 응답, 답변은 스레드로 격리 |

### 설정이 두 경로로 갈린다 (중요)

Discord 설정은 키마다 읽는 경로가 다르다. 어느 쪽인지 모르고 건드리면 반영이 안 된다.

| 경로 | 키 | 바꾸는 곳 |
|---|---|---|
| `os.getenv` 직접 | `allowed_users`, `allowed_channels`, `ignored_channels`, `no_thread_channels`, `auto_thread`, `allow_bots`, `home_channel` | **Git** — [deployment.yaml](manifests/deployment.yaml) 의 `DISCORD_*` env |
| `config.extra` 우선 | `require_mention`, `free_response_channels`, `thread_require_mention`, `history_backfill(_limit)`, `allow_any_attachment`, `max_attachment_bytes`, `slash_commands` | **대시보드** — https://hermes.json-server.win |

env 경로는 컨테이너 환경변수라 대시보드에서 못 바꾼다. 즉 **누가 부를 수 있고 어느 채널까지
열려 있는지는 Git 이 강제**하며, 대시보드로 느슨하게 만들 수 없다. 여기가 보안 경계다.

`config.extra` 경로는 반대다. getter 가 이렇게 생겼다:

```python
raw = self.config.extra.get("free_response_channels")
if raw is None:
    raw = os.getenv("DISCORD_FREE_RESPONSE_CHANNELS", "")
```

라이브 config 에 키가 **빈 문자열로 존재**하면 `None` 이 아니므로 env 를 아예 보지 않는다.
config→env 다리(`if cfg is not None and not os.getenv(...)`)는 env 가 비었을 때 config 로 env 를
채울 뿐이라 이 경로를 구제하지 못한다. **그래서 이 키들에 `DISCORD_*` env 를 걸어도 무의미하다.**

이 키들은 Hermes 가 PVC 위 `config.yaml` 의 주인이므로 **대시보드에서 관리한다.** seed
initContainer 는 라이브 파일을 덮지 않으니 Git 의 [config.yaml](manifests/config.yaml) 은
PVC 가 비었을 때의 최초 시드일 뿐이다.

### 보안 기본값 (주의)

- `DISCORD_ALLOWED_USERS` 가 비면 **fail-open** — 어댑터 원문: *"If both allowlists are empty,
  everyone is allowed"*. 봇이 들어간 서버의 아무나, DM 으로도 셸 실행이 가능한 에이전트를
  부릴 수 있다. **비운 채로 배포 금지**.
- `DISCORD_ALLOW_BOTS=mentions` — Alertmanager 는 웹훅이라 discord.py 기준 `author.bot=True`.
  기본값 `none` 이면 on_message 차단은 물론 **히스토리 백필에서도 제외**돼 알람 본문을 아예 못
  읽는다(`include_other_bots = allow_bots_raw != "none"`). `all` 은 알람마다 자동 응답해 토큰이
  폭주한다. `mentions` 만이 "읽기는 되고 깨우지는 않는" 조합이다.
- 봇 초대 권한은 읽기/쓰기/스레드까지만. Manage 계열·Administrator 미부여.

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
  `DISCORD_BOT_TOKEN`.

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
- Discord 봇 토큰 ([Developer Portal](https://discord.com/developers/applications), `MESSAGE CONTENT INTENT` 필수)
- 본인 Discord 유저 ID (개발자 모드 → 우클릭 → ID 복사) — `DISCORD_ALLOWED_USERS`

---

## Phase A — 봉인 + 배포

cert는 `k8s/sealed-secrets/cert.pem`에 커밋되어 있어 VPN/클러스터 접근 없이 sealing 가능.

**auth.json 봉인**: `cat ~/hermes-codex/auth.json | KUBECONFIG=~/.kube/config-json kubeseal --raw
--cert k8s/sealed-secrets/cert.pem --name hermes-secrets --namespace hermes --scope strict`
→ `sealed-secret.yaml`의 `auth.json` 키에 넣음. admin/API_SERVER_KEY/DASHBOARD_SECRET은 봉인 완료.
→ PR 머지하면 ArgoCD가 `apps/hermes.yaml` 자동 sync.

Discord 봇 토큰 교체:
```bash
seal() { KUBECONFIG=~/.kube/config-json kubeseal --raw --cert k8s/sealed-secrets/cert.pem \
  --name hermes-secrets --namespace hermes --scope strict; }
echo -n "$DISCORD_BOT_TOKEN" | seal   # → sealed-secret.yaml encryptedData.DISCORD_BOT_TOKEN
kubectl -n hermes rollout restart deploy/hermes   # optional env가 키를 집음
```

## 검증 (post-merge, hard-refresh 먼저)

```bash
kubectl -n argocd patch app hermes --type=merge \
  -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}'
argocd app wait hermes --sync --health --timeout=600
# SealedSecret 복호화 확인 (empty 함정):
kubectl -n hermes get secret hermes-secrets -o json | jq '.data | map_values(@base64d | length)'
kubectl -n hermes logs deploy/hermes -c gateway | grep -i "discord\|provider"
```
- Codex(ChatGPT 구독)로 추론 동작 (subscription_included, 종량제 미사용)
- `hermes.json-server.win` → Authentik 통과 후 대시보드 (admin: hermes / 봉인된 pass)
- Discord `#일반`에서 멘션 없이 대화 왕복, `#alert`에서 `@Hermes` 멘션 시 최근 알람 인용 응답

## 운영 메모

- 모델 변경: 대시보드 또는 파드 내 `hermes config set model.default <id>`. `config.yaml`(ConfigMap)은
  seed 전용 — 라이브 파일을 덮지 않음.
- 이미지 핀: 제3자 공개 이미지 → 수동 태그 업데이트 ([manifests/deployment.yaml](manifests/deployment.yaml)
  의 `v2026.6.5`). Image Updater 미사용.
- 백업: `/opt/data` = 기억/스킬/세션/인증 전부. PVC 유실 시 seed에서 재구성하되 OAuth는 재로그인 필요.
