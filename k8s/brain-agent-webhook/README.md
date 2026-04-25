# brain-agent-webhook

Telegram 양방향 HTTP 서버 — 장수님 텔레그램 메시지를 수신해서 health-hub에 저장.

## 기능

| 입력 | 동작 |
|---|---|
| 텍스트 메시지 | `POST /api/v1/notes` (category: `tg-text`) |
| 사진 메시지 | OpenAI gpt-4o Vision 분석 → inline button 확인 → `POST /api/v1/meals` |
| `/ping` | pong |

## 아키텍처

```
[장수님 핸드폰] ─→ Telegram ─HTTPS POST→ brain-agent.json-server.win/webhook/telegram
                                              ↓
                                     FastAPI (automation ns)
                                              ↓
                                  OpenAI gpt-4o Vision (사진 분석)
                                              ↓
                         health-hub REST (/api/v1/notes, /api/v1/meals)
                                              ↓
                                        Telegram 응답
```

## Stateless 설계

callback_data 형식: `action|k=v|k=v` (Telegram 64 byte 제한)
예: `save_meal|t=lun|c=820|p=35|cb=90|f=30`

식사 이름은 confirmation message 본문 첫 줄에서 파싱 (이모지 제거 후).

## 배포 후 초기 설정 (한 번만)

### 1. Telegram setWebhook 호출

```bash
BOT_TOKEN=$(kubectl get secret alertmanager-telegram -n observability \
  -o jsonpath='{.data.bot-token}' | base64 -d)
SECRET=$(kubectl get secret brain-agent-tokens -n automation \
  -o jsonpath='{.data.WEBHOOK_SECRET}' | base64 -d)

curl -sG -X POST "https://api.telegram.org/bot${BOT_TOKEN}/setWebhook" \
  --data-urlencode "url=https://brain-agent.json-server.win/webhook/telegram" \
  --data-urlencode "secret_token=${SECRET}" \
  --data-urlencode 'allowed_updates=["message","callback_query"]'
```

응답 `{"ok":true,"result":true,"description":"Webhook was set"}` 확인.

### 2. 헬스체크

```bash
curl -s https://brain-agent.json-server.win/healthz
# {"ok":true}
```

### 3. 실전 테스트

1. 텔레그램 장수님 thread에 "테스트" 텍스트 전송 → `✅ 노트 저장됨` 응답
2. 식사 사진 전송 → `🔍 분석 중...` → 영양소 카드 + 버튼 → 저장 확인

## 환경변수

| 이름 | 의미 | 출처 |
|---|---|---|
| `TELEGRAM_BOT_TOKEN` | Bot API 토큰 | observability/alertmanager-telegram (Reflector) |
| `TELEGRAM_CHAT_ID` | 허용된 채팅 ID | observability/alertmanager-telegram (Reflector) |
| `TELEGRAM_THREAD_ID` | 포럼 스레드 ID (206) | observability/alertmanager-telegram (Reflector) |
| `WEBHOOK_SECRET` | Telegram X-Telegram-Bot-Api-Secret-Token 검증 | automation/brain-agent-tokens |
| `OPENAI_API_KEY` | OpenAI gpt-4o Vision | automation/brain-agent-tokens |
| `OPENAI_VISION_MODEL` | 모델 (기본 `gpt-4o`, 환경변수로 override 가능) | env value |
| `HEALTH_HUB_URL` | `https://health.json-server.win` | env value |
| `HEALTH_HUB_TOKEN` | health-hub REST API_TOKEN | health-hub/health-hub-api-token (Reflector) |

## 비용 예측

- 사진 분석 (gpt-4o Vision): ~$0.006/장
- 하루 2장 × 30일 = ~$0.33/월
- 텍스트·callback 처리: $0

## 향후 확장

- 저녁 컨디션 체크인 CronJob (예: 22시에 질문 → 사용자 답변 수신)
- `/query` 명령어로 brain 검색 (gbrain MCP 경유)
- 수정 (edit_meal) 기능 v2
- 품질 부족 시 Claude Sonnet Vision 옵션 추가 (현재 비용 30배 차이로 미사용)
