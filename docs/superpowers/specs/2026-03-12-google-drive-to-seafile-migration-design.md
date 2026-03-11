# Google Drive → Seafile Migration Design

## Overview

Google One 요금제 탈퇴를 위해 Google Drive 데이터를 셀프호스팅 Seafile로 마이그레이션한다.
이전 Google Photos → Immich 마이그레이션에 이어 두 번째 단계.

## Constraints

- 홈랩 외부 인터넷: 100Mbps
- 홈랩 내부 LAN: 1Gbps
- rclone Google Drive remote 인증 완료 (테스트 통과)

## Source

- **Google Drive**: ~82GB (rclone size 기준, 23,510 파일)
- **Google 네이티브 파일**: 9개 (크기 미포함, docx/xlsx/pptx로 자동 변환)
- **참고**: Google Forms, Sites는 rclone export 대상 아님. Drawings는 PNG/SVG로 변환. 댓글/제안 히스토리 손실.
- **OAuth client**: `openclaw-json` 프로젝트 credentials 재사용 (client_id: `424776597177-...`)

## Target

- **Seafile**: files.json-server.win
- **라이브러리**: 단일 라이브러리에 전체 업로드
- **데이터 PV**: `/mnt/hdd-seagate-1t/seafile-data` (hostPath, 100Gi)

## Storage Plan

| 항목 | 위치 | 용량 |
|------|------|------|
| Seafile 데이터 | `/mnt/hdd-seagate-1t/seafile-data/` | 기존 + ~82GB |
| Immich 데이터 | `/mnt/hdd-seagate-1t/immich-library/` | ~272GB |
| Frigate 미디어 | `/mnt/hdd-seagate-1t/frigate-media/` | PV 200Gi 할당 (실 사용량 확인 필요) |

Seagate 1TB (931.5GB) — 마이그레이션 전 `df -h /mnt/hdd-seagate-1t`로 실제 여유 공간 확인 필수.
Seafile PV 100Gi 제한 — 기존 Seafile 사용량 + 82GB가 100Gi 이내인지 확인. 초과 시 PV 확장 필요.

> **임시 저장 공간 불필요**: rclone이 Google Drive → Seafile 직접 전송하므로 중간 저장 없음.

## Migration Steps

### 1. 사전 준비

```bash
# 서버 디스크 여유 확인
ssh json@192.168.0.27 "df -h /mnt/hdd-seagate-1t"

# 기존 Seafile 사용량 확인
ssh json@192.168.0.27 "du -sh /mnt/hdd-seagate-1t/seafile-data/"
```

Seafile PV 100Gi 부족 시 → PV/PVC 크기 증가 (hostPath라 실질적 제한 없음, manifest만 수정).

### 2. rclone 설정

Google Drive remote는 로컬 PC에 이미 설정 완료 (`gdrive:`).

```bash
# Seafile remote 설정
rclone config create seafile seafile \
  url="https://files.json-server.win" \
  user="manamana32321@gmail.com" \
  pass="$(rclone obscure '<seafile-password>')" \
  library="Google Drive" \
  create_library=true
```

### 3. Seafile 라이브러리 생성 + 전송

```bash
# dry-run으로 먼저 확인
rclone copy gdrive: seafile: \
  --drive-export-formats docx,xlsx,pptx \
  --progress --transfers 4 --checkers 8 \
  --dry-run

# 실제 전송
rclone copy gdrive: seafile: \
  --drive-export-formats docx,xlsx,pptx \
  --progress --transfers 4 --checkers 8
```

- rclone이 Google Drive 전체를 Seafile 라이브러리에 직접 복사
- Google 네이티브 파일은 Office 형식으로 자동 변환
- 실패 시 자동 재시도 (기본 3회)
- 100Mbps 외부 인터넷 기준 ~82GB → 약 110분 (Google API throttling에 따라 변동)

### 4. 검증

```bash
# Google Drive 파일 수/크기
rclone size gdrive:

# Seafile 라이브러리 파일 수/크기
rclone size seafile:

# 차이 확인
rclone check gdrive: seafile: \
  --drive-export-formats docx,xlsx,pptx \
  --one-way
```

- `rclone check`로 소스-대상 파일 일치 여부 자동 검증
- Seafile 웹 UI에서 주요 파일 열어서 내용 확인
- 특히 Office 변환된 네이티브 파일 서식 확인

### 5. 정리

- rclone config에서 `gdrive` remote 제거 (선택)
- OAuth client credentials는 보관 (다른 용도 재사용 가능)

### 6. Google One 해지

- 마이그레이션 검증 완료 후 당분간 유지
- Google One 해지 후에도 유예 기간 있음 (15GB 초과 데이터 즉시 삭제 안 됨)
- 충분히 검증 후 Drive 데이터 정리 및 Google One 해지

## Risks & Mitigations

| 리스크 | 대응 |
|--------|------|
| Google API rate limit | rclone 기본 rate limiting 준수, `--tpslimit` 조정 가능 |
| Office 변환 시 서식 깨짐 | 9개 파일이라 수동 확인 가능 |
| Seafile WebDAV 업로드 느림/타임아웃 | `--transfers` 수 줄이기, `--timeout` 증가 |
| 네트워크 끊김 | rclone 자동 재시도, 재실행 시 이미 전송된 파일 스킵 |
| Seafile PV 100Gi 초과 | 사전 확인, 필요 시 PV manifest 수정 |
| Google Forms/Sites 누락 | rclone export 대상 아님, 별도 수동 백업 필요 시 확인 |

## Timeline

1. 사전 확인 (디스크, Seafile 사용량): ~5분
2. rclone Seafile remote 설정: ~5분
3. dry-run 확인: ~5분
4. 실제 전송 (100Mbps 기준 82GB): ~110분
5. 검증 (`rclone check` + 수동 확인): ~10분

**총 예상 소요: ~2시간 15분**

## Dependencies

- rclone 설치 완료 (로컬 PC, v1.69.2)
- Google Drive OAuth 인증 완료 (`gdrive:` remote)
- Seafile 정상 동작 (files.json-server.win)
- Seafile 계정 비밀번호
