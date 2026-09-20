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

## 클러스터 접근 (kubernetes-mcp)

에이전트가 알람을 보고 원인까지 짚으려면 클러스터를 읽어야 한다. **Hermes 파드에는
클러스터 크레덴셜을 두지 않는다** — `automountServiceAccountToken: false` 로 SA 토큰
마운트 자체를 끈다. 대신 같은 네임스페이스에 [kubernetes-mcp](manifests/mcp-kubernetes.yaml)
를 띄우고, Hermes 의 내장 MCP 클라이언트가 HTTP 로 붙는다.

```
Hermes (토큰 없음)  ──MCP/HTTP──▶  kubernetes-mcp  ──SA 토큰──▶  kube-apiserver
                                   (RBAC 여기에)
```

권한이 Hermes 프로세스 밖에 있으므로, 프롬프트 인젝션이 나도 **MCP 서버가 노출한 도구
밖으로는 나갈 수 없다.** 에이전트가 토큰을 읽어 임의 API 를 호출하는 경로가 없다.

### MCP 서버

- 이미지 `quay.io/containers/kubernetes_mcp_server` ([containers/kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server), Apache-2.0)
- `port = "8080"` → Streamable HTTP 가 `/mcp` 에 뜬다
- `toolsets = ["core"]` — helm/kubevirt/tekton 등은 붙이지 않는다
- `denied_resources` 로 `v1/Secret` 차단 (RBAC 에 더해 앱 계층 한 겹)
- Helm 차트(`ghcr.io/containers/charts/kubernetes-mcp-server`)는 쓰지 않는다. 렌더링해 보면
  SA/ConfigMap/Service/Deployment 4개만 나오고 **RBAC 은 생성되지 않으며**, 이미지가
  `latest` 로 고정돼 있고 `ingress.enabled: true` 가 기본이라 host 없이는 렌더링이 실패한다.
  정작 필요한 권한·읽기전용·Secret 차단은 전부 차트 밖이라 직접 쓰는 편이 짧다.

### 네트워크 경계

MCP 서버는 인증·TLS 없이 `0.0.0.0:8080` 에 바인딩한다(기동 로그에 경고가 찍힌다).
ClusterIP 라 외부 노출은 없지만, **클러스터 안 아무 파드나 호출하면 그 RBAC 을 그대로
빌려 쓸 수 있다.** 실측으로 확인했다 — `default` 네임스페이스의 임시 파드가
`/healthz` 에 HTTP 200 으로 닿았다.

그래서 NetworkPolicy 로 ingress 를 Hermes 파드로만 제한한다. 적용 후 재측정:

| 출발지 | 적용 전 | 적용 후 |
|---|---|---|
| `default` ns 임의 파드 | HTTP 200 | **차단** |
| Hermes 파드 | HTTP 200 | HTTP 200 |
| kubelet probe | 정상 | 정상 (차단되지 않음) |

kubelet probe 가 NetworkPolicy 에 막히면 파드가 CrashLoop 에 빠지므로 적용 전에 90초간
확인했다. 이 클러스터(k3s + flannel + 내장 kube-router netpol 컨트롤러)에서는 막히지 않는다.

### 권한 (MCP 서버의 SA 에 붙는다)

| 범위 | 내용 |
|---|---|
| 읽기 (클러스터 전체) | 빌트인 `view` + `kubernetes-mcp-read-infra` |
| 쓰기 (10개 ns) | `kubernetes-mcp-workload-restart` 를 RoleBinding 으로 부착 |

빌트인 `view` 는 `secrets`·`pods/exec`·`pods/portforward`·`pods/attach` 를 포함하지 않는다
(실측 확인). 다만 `nodes`·`persistentvolumes`·`longhorn.io` CR 도 빠져 있어, 이 홈랩의
주된 장애(노드 NotReady, DiskPressure, Longhorn faulted)를 진단할 수 없다. 그래서
`kubernetes-mcp-read-infra` 로 그 셋만 읽기 전용으로 보탠다.

쓰기는 재시작 계열로 제한한다 — `patch` (deployments/statefulsets/daemonsets 와 그
`scale` 서브리소스), `delete pods`. 앱 자체를 지우거나 만들 수는 없다.

쓰기 대상 ns: `immich` `seafile` `home-assistant` `minecraft` `mosquitto` `nightscout`
`gbrain` `hermes` `health-hub` `observability`.
**제외**: `kube-system` `argocd` `cert-manager` `authentik` `longhorn-system`
`sealed-secrets` `amang-*` `essentia`.

## gbrain MCP

개인 지식(brain) 검색을 에이전트에 붙인다. 별도 파드를 띄우지 않는다 — `gbrain-mcp` 가
이미 클러스터에서 돌고 있으므로 **내부 ClusterIP 로 직접 붙는다**(Cloudflare 헤어핀 회피).

```
Hermes ──SSE──▶ http://gbrain-mcp.gbrain.svc.cluster.local/sse
                  └ 파드 내 Caddy 가 Bearer 게이트. 내부 접근도 401 로 막힌다(실측)
```

토큰은 **설정 파일에 평문으로 넣지 않는다.** Hermes MCP 설정은 `${VAR}` 치환을 지원하므로
(`mcp_config.py` 의 `_resolve_mcp_server_config`), 헤더에는 플레이스홀더만 두고 실제 값은
컨테이너 env 로 주입한다:

```yaml
mcp_servers:
  gbrain:
    url: http://gbrain-mcp.gbrain.svc.cluster.local/sse
    transport: sse
    headers:
      Authorization: "Bearer ${GBRAIN_MCP_TOKEN}"
```

`GBRAIN_MCP_TOKEN` 은 SealedSecret `hermes-secrets` 의 동명 키에서 온다. 원본은 gbrain
네임스페이스의 `gbrain-postgres-secrets/MCP_AUTH_TOKEN` 이고, 네임스페이스가 달라 직접
참조할 수 없으므로 hermes 쪽에 별도로 봉인했다. gbrain 쪽 토큰이 바뀌면 여기도 재봉인해야
한다.

> **쓰기 도구는 쓰지 않는다.** brain 의 SSOT 는 markdown 파일 + git 이고, MCP write 도구
> (`put_page` 등)는 DB 만 갱신해 orphan 을 만든다. 에이전트에는 읽기 도구만 노출한다.

### Hermes 쪽 연결

`mcp_servers` 는 PVC 위 라이브 `config.yaml` 에 있다(대시보드/런타임 관리 영역):

```yaml
mcp_servers:
  kubernetes:
    url: http://kubernetes-mcp.hermes.svc.cluster.local:8080/mcp
    trust: untrusted
```

### 확인 / 회수

```bash
SA=system:serviceaccount:hermes:kubernetes-mcp
kubectl auth can-i list nodes                 --as=$SA           # yes
kubectl auth can-i get secrets                --as=$SA -A        # no
kubectl auth can-i create pods/exec           --as=$SA -A        # no
kubectl auth can-i patch deployments          --as=$SA -n immich # yes
kubectl auth can-i patch deployments          --as=$SA -n argocd # no

# Hermes 파드에는 토큰이 없어야 한다
kubectl -n hermes exec deploy/hermes -c gateway -- ls /var/run/secrets/kubernetes.io/ 2>&1
```

권한을 되돌리려면 [rbac.yaml](manifests/rbac.yaml) 에서 해당 바인딩을 지우고,
접근을 완전히 끊으려면 [mcp-kubernetes.yaml](manifests/mcp-kubernetes.yaml) 을 지우면 된다.

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
