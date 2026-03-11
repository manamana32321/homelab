# Google Drive → Seafile Migration Design

## Overview

Google One 요금제 탈퇴를 위해 Google Drive 데이터를 셀프호스팅 Seafile로 마이그레이션한다.
이전 Google Photos → Immich 마이그레이션에 이어 두 번째 단계.

## Constraints

- VPN 없음 (원격) — 집에서 LAN 접근 시 실행
- 홈랩 외부 인터넷: 100Mbps
- 홈랩 내부 LAN: 1Gbps
- Google Takeout 이미 요청 완료 (10GB 분할)

## Source

- **Google Drive**: ~41GB 사용 중
- **Google 네이티브 파일**: 20개 미만 (Docs/Sheets/Slides → Office 형식 변환)
- **참고**: Google Forms, Sites는 Takeout에 포함 안 됨. Drawings는 PNG/SVG로 변환. 댓글/제안 히스토리 손실.

## Target

- **Seafile**: files.json-server.win
- **라이브러리**: 단일 라이브러리에 전체 업로드
- **데이터 PV**: `/mnt/hdd-seagate-1t/seafile-data` (hostPath, 100Gi)

## Storage Plan

| 항목 | 위치 | 용량 |
|------|------|------|
| 임시 저장 (zip + 해제) | `/mnt/hdd-seagate-1t/google-drive-tmp/` | ~82GB (peak) |
| Seafile 데이터 | `/mnt/hdd-seagate-1t/seafile-data/` | 기존 + ~41GB |
| Immich 데이터 | `/mnt/hdd-seagate-1t/immich-library/` | ~272GB |
| Frigate 미디어 | `/mnt/hdd-seagate-1t/frigate-media/` | PV 200Gi 할당 (실 사용량 확인 필요) |

Seagate 1TB (931.5GB) — 마이그레이션 전 `df -h /mnt/hdd-seagate-1t`로 실제 여유 공간 확인 필수.
Seafile PV 100Gi 제한 — 기존 Seafile 사용량 + 41GB가 100Gi 이내인지 확인 필요.

## Migration Steps

### 1. 사전 준비
- 서버 디스크 여유 확인: `df -h /mnt/hdd-seagate-1t`
- 기존 Seafile 사용량 확인: `du -sh /mnt/hdd-seagate-1t/seafile-data/`
- 임시 디렉토리 생성: `mkdir -p /mnt/hdd-seagate-1t/google-drive-tmp`

### 2. Takeout 다운로드

Google Takeout 링크는 인증 쿠키가 필요하므로 단순 `wget`으로 안 됨.

**방법: 로컬 PC 다운로드 → rsync로 서버 전송**
```bash
# 로컬 PC에서 브라우저로 Takeout zip 다운로드 (이미 완료 가능)
# 집에서 LAN으로 서버에 전송
rsync -avP --bwlimit=0 ~/Downloads/takeout-*.zip \
  json@192.168.0.27:/mnt/hdd-seagate-1t/google-drive-tmp/
```
LAN 1Gbps 기준 41GB → ~6분

### 3. 압축 해제
```bash
cd /mnt/hdd-seagate-1t/google-drive-tmp/

# 10GB 분할 zip — 같은 디렉토리에 해제 (폴더 구조 자동 병합)
for f in *.zip; do unzip -o "$f" -d extracted/; done

# zip 삭제하여 디스크 절약
rm -f *.zip
```
> **주의**: 분할 zip은 같은 폴더의 파일이 여러 zip에 나뉠 수 있음. `-o` 플래그로 덮어쓰기 허용. 해제 후 파일 수로 검증.

### 4. 폴더 구조 확인
```bash
# Takeout 구조: Takeout/Drive/... 형태
ls extracted/

# 파일 수 확인
find extracted/ -type f | wc -l

# 총 크기 확인
du -sh extracted/

# 디렉토리 구조 확인
tree -d -L 3 extracted/
```

### 5. Seafile 라이브러리 생성
- Seafile 웹 UI (files.json-server.win)에서 새 라이브러리 생성
- 이름: "Google Drive" 또는 원하는 이름

### 6. 파일 업로드

**방법 A: rclone (권장)**

rclone은 Seafile remote 타입을 네이티브 지원. 재귀 업로드, 자동 재시도 포함.

```bash
# rclone 설치 (없으면)
curl https://rclone.org/install.sh | sudo bash

# Seafile remote 설정
rclone config
# type: seafile
# url: https://files.json-server.win
# user: <email>
# pass: <password>
# library: <library-name>
# create_library: false

# 업로드 (Takeout/Drive/ 하위를 라이브러리 루트로)
rclone copy extracted/Takeout/Drive/ seafile: \
  --progress --transfers 4 --checkers 8
```

**방법 B: Seafile Web API**
```bash
# 인증 토큰 획득
TOKEN=$(curl -s -d "username=<email>&password=<password>" \
  https://files.json-server.win/api2/auth-token/ | jq -r .token)

# 디렉토리 생성
curl -X POST -H "Authorization: Token $TOKEN" \
  "https://files.json-server.win/api2/repos/<repo-id>/dir/" \
  -d "p=/<dir-path>"

# 업로드 링크 획득
UPLOAD_LINK=$(curl -s -H "Authorization: Token $TOKEN" \
  "https://files.json-server.win/api2/repos/<repo-id>/upload-link/" | jq -r .)

# 파일 업로드 (디렉토리별 parent_dir 지정 필요)
curl -H "Authorization: Token $TOKEN" \
  -F "file=@<filepath>" -F "parent_dir=/<dir>" \
  "$UPLOAD_LINK"

# 전체 자동화는 재귀 스크립트 필요 — rclone이 안 되면 이 방법 사용
```

> **seaf-cli는 사용하지 않음**: 양방향 동기화 도구라 임시 파일 삭제 시 서버 데이터도 삭제됨 (데이터 손실 위험). 일회성 마이그레이션에 부적합.

### 7. 검증
- Seafile 웹에서 파일 목록 확인
- 파일 수 비교: `find extracted/Takeout/Drive/ -type f | wc -l` vs Seafile 라이브러리
- 총 크기 비교: `du -sh extracted/Takeout/Drive/` vs Seafile 라이브러리 속성
- 주요 파일 열어서 내용 확인 (특히 Office 변환된 네이티브 파일)
- Google Drive 웹과 대조

### 8. 정리
```bash
rm -rf /mnt/hdd-seagate-1t/google-drive-tmp/
```

### 9. Google One 해지
- 마이그레이션 검증 완료 후 당분간 유지 (C안)
- Google One 해지 후에도 유예 기간 있음 (15GB 초과 데이터 즉시 삭제 안 됨)
- 충분히 검증 후 Drive 데이터 정리 및 Google One 해지

## Risks & Mitigations

| 리스크 | 대응 |
|--------|------|
| Takeout 링크 만료 (7일) | 만료 전 다운로드 실행, 이미 로컬 다운 가능 |
| Office 변환 시 서식 깨짐 | 20개 미만이라 수동 확인 가능 |
| rclone 설치/설정 실패 | Web API 스크립트로 대체 |
| 업로드 중 네트워크 끊김 | rclone 자동 재시도 |
| 분할 zip 해제 시 파일 누락 | 해제 후 파일 수/크기로 검증 |
| Seafile PV 100Gi 초과 | 사전에 기존 사용량 확인 |
| Google Forms/Sites/Drawings 누락 | Takeout 전 네이티브 파일 목록 확인 (20개 미만) |

## Timeline

1. 사전 확인 (디스크, Seafile 사용량): ~5분
2. Takeout zip → 서버 전송 (LAN rsync): ~6분
3. 압축 해제 + zip 삭제: ~5분
4. rclone 설정 + 업로드: ~15분
5. 검증: ~10분
6. 임시 파일 정리: ~1분

**총 예상 소요: ~40분** (집에서 LAN 기준)

## Dependencies

- 서버 SSH 접속 (집에서 LAN)
- Seafile 정상 동작 확인
- rclone 설치 가능 (또는 curl/jq for Web API)
