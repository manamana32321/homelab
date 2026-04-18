# OpenClaw State Archive Location

OpenClaw 런타임 상태([`openclaw-home` PVC](./backup-cronjob.yaml))의 로컬 아카이브. Claude 마이그레이션([plan](../../../../../.claude/plans/bright-growing-naur.md))의 일회성 증류 소스 — Phase 4 완료 후 deprecated 예정 (Intent 섹션 참조).

## Location

- **Absolute path**: `~/openclaw-archive/openclaw-state/`
- **Upstream**: [manamana32321/openclaw-state](https://github.com/manamana32321/openclaw-state) (private)
- **Clone method**: `gh repo clone` (HTTPS + `manamana32321` GitHub 계정 토큰)

## Why outside homelab repo

- 878MB (media/ 569MB + agents/ 75MB + .git 159MB) — git push 부담
- `agents/*/sessions/*.jsonl`이 평문 — 선언적 인프라 관리 관심사와 분리
- homelab repo의 public 전환 가능성 대비 민감 데이터 격리

## Read-only lockdown

`agents/*/sessions/` 디렉토리는 `chmod -R a-w`로 잠금. 스크립트 오작동/실수로부터 원본 보호.

검증:

```bash
touch ~/openclaw-archive/openclaw-state/agents/main/sessions/test.txt
# Permission denied 기대
```

## Sync status

OpenClaw 사용 중단으로 upstream 갱신 없음. 본 archive가 **마지막 스냅샷**이며 `git pull` 불필요.

## Retained vs Discarded

Phase 4 후 deprecated 예정이라 archive의 현재 용도는 **Claude 증류 소스 + 참조용 문서 보존**뿐. 디렉토리별 처리:

### 보존 (증류·참조 대상)

- `agents/*/sessions/*.jsonl` — Claude 증류 소스 (main 제외, Morton 회사 비서라 분리)
- `media/inbound/*.{txt,pdf,md}` — 업로드 문서류 (예: VIA Character Strengths Survey). 증류 시 세션 참조와 연결
- `media/inbound/*.json` — 2개, 내용 확인 후 문서성 여부 판단
- `openclaw.json` — 에이전트/채널 설정 원본. 별도 재구축 참조용

### 폐기 (증류 불필요, Phase 2 완료 후 일괄 정리)

- `media/inbound/*.{ogg,jpg,png,m4a}` — Telegram 음성/이미지 (569MB). 텍스트 메타는 sessions에 이미 포함
- `memory/*.sqlite` — RAG/FTS 인덱스. 런타임 없으면 재인덱싱 불필요
- `tasks/*.sqlite*`, `flows/*.sqlite*` — OpenClaw 내부 운영 상태
- `credentials/`, `identity/`, `devices/`, `telegram/` — 채널 페어링·폴링 상태
- `.env`, `google-*.json`, `exec-approvals.json`, `restart-sentinel.json`, `update-check.json`, `agents/*/agent/auth-*.json` — 토큰·운영 상태 (토큰 폐기/교체 예정)

## Intent

- 이 archive는 **Claude 마이그레이션([plan](../../../../../.claude/plans/bright-growing-naur.md))의 증류 소스 전용**이다.
- Phase 4 완료(OpenClaw → Claude 이전 검증) 후 **이 archive와 본 파일 모두 deprecated** — 마지막 cleanup PR에서 함께 제거.
- `agents/main/`(래미)은 **Morton 회사 비서용**이라 본 개인 Claude memory에 증류하지 않는다. 회사 비서는 별도 세션·레포에서 처음부터 재구축 예정.
