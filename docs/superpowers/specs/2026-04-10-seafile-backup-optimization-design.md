# Seafile 백업 최적화 + 라이브러리 통합

> Issue: [manamana32321/homelab#83](https://github.com/manamana32321/homelab/issues/83)
> Date: 2026-04-10 (최종 수정: 2026-04-11)
> PRs: #89 (merged, 초기 전략), #96 (merged, 최종 전략)

## 배경

### 문제 1: Glacier IR 요청 비용 폭탄

3월 AWS 청구서 **$113.52** 중 S3 요청 비용이 **$99.75**.

- rclone sync가 매일 Seafile 블록 파일 506K 오브젝트에 HEAD 요청
- Glacier IR GET/HEAD 요청 단가: **$0.01/1K** (Standard의 25배)
- `--fast-list`로도 HEAD 제거 불가 (실측 확인: 3/22 적용 후에도 비용 미감소)

### 문제 2: Seafile 버전 히스토리 오브젝트 폭발

- 49만 오브젝트 중 46만이 commits/fs (버전 히스토리)
- `keep_days=0` 설정으로 새 히스토리 방지 + 라이브러리 재생성으로 기존 히스토리 제거

### 문제 3: 강의 영상 백업 불필요

LearningX에서 재다운로드 가능한 강의 영상을 S3에 백업할 필요 없음.

## 최종 설계

### S3 스토리지 클래스: Intelligent-Tiering

초기 전략(Standard→lifecycle→GIR)에서 변경. lifecycle 전환 후에도 GIR 오브젝트에 HEAD 요청 시 GIR 요금이 적용되므로 근본 해결이 안 됨.

IT 선택 이유:
- HEAD 요청이 **Standard 요금** ($0.005/1K)
- 90일 후 Archive Instant Access = **GIR과 동일 저장 비용** ($0.005/GB)
- **복원 비용 $0** (GIR은 $0.01/GB)
- 모니터링 비용: $0.0025/1K objects/월 (미미)

### Seafile 라이브러리 구조 (PARA)

| 라이브러리 | UUID | 용도 | 백업 |
|-----------|------|------|------|
| 내 라이브러리 | `66a1e14f-8c91-45c8-ace8-b3c7f03ac63e` | PARA 구조 개인 파일 | IT |
| 녹음 | `32e4bb08-fc91-4ddd-992e-a6da1bb814fd` | 통화/음성 녹음 | IT |
| 강의영상 | `07c9cf44-8874-491b-9767-ef8783d407f5` | 강의 영상 전용 | 제외 |

### rclone 설정

**Seafile**: include 필터로 2개 라이브러리만 sync
```
storage_class = INTELLIGENT_TIERING
--filter "+ seafile/seafile-data/storage/*/66a1e14f-.../**"
--filter "+ seafile/seafile-data/storage/*/32e4bb08-.../**"
--filter "- **"
```

**Immich**: 기존 GIR + 새 파일 IT 혼합
```
storage_class = INTELLIGENT_TIERING
--s3-no-head  (기존 GIR 오브젝트 HEAD 비용 방지)
```

### 비용

| | Before (3월) | After |
|---|---|---|
| 월간 | $113.52 | ~$2.50 |

## 완료된 작업

- [x] Seafile `keep_days=0` 설정 (버전 히스토리 비활성화)
- [x] 라이브러리 재생성 (히스토리 제거, 494K→60K 오브젝트)
- [x] PARA 폴더 구조 정리
- [x] 녹음 별도 라이브러리 분리
- [x] Google Drive 중복 폴더 삭제
- [x] S3 기존 Seafile 데이터 삭제
- [x] Terraform lifecycle rule 제거
- [x] CronJob rclone IT 설정 + include 필터
- [x] terraform apply
- [x] 디스크 고아 블록 정리
- [x] CronJob suspend 해제

## 미완료

- [ ] 첫 sync 결과 확인
- [ ] (2026-06-24 이후) Immich GIR→IT COPY 전환 ($0.42)
