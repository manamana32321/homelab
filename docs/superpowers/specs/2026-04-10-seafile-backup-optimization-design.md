# Seafile 백업 최적화 + 라이브러리 통합

> Issue: [manamana32321/homelab#83](https://github.com/manamana32321/homelab/issues/83)
> Date: 2026-04-10

## 배경

### 문제 1: Glacier IR 요청 비용 폭탄

3월 AWS 청구서 **$113.52** 중 S3 요청 비용이 **$99.75**.

- rclone sync가 매일 Seafile 블록 파일 506K 오브젝트를 HEAD 요청으로 비교
- Glacier IR GET/HEAD 요청 단가: **$0.01/1K** (Standard의 25배)
- 506K obj × 30일 = ~15M 요청 → $87.75 (GIR-Tier2) + $12 (GIR-Tier1)

### 문제 2: 강의 영상 백업 불필요

LectureHub 라이브러리에 저장될 강의 영상은 LearningX에서 재다운로드 가능.
수백GB가 될 수 있어 백업 대상에서 제외 필요.

### 문제 3: 라이브러리 난립

현재 4개 라이브러리가 용도 구분 없이 산재.

| 라이브러리 | 용량 | 내용 |
|-----------|------|------|
| 내 라이브러리 | 18GB | Recordings/, 2026-1학기/, 건강/ |
| Google Drive | 41GB | 구글 드라이브 마이그레이션 전체 |
| LectureHub | 232KB | 2026-1/ 과목별 과제 파일 |
| My Library Template | 0 | 빈 템플릿 |

## 설계

### 1. S3 Storage Class 전략 변경

**원칙**: rclone은 Standard에 sync, lifecycle rule이 Glacier IR로 전환.

#### rclone 변경

Immich/Seafile media backup CronJob에서:

```diff
- storage_class = GLACIER_IR
+ # storage_class 제거 — Standard default
```

#### S3 Lifecycle Rule 추가

각 백업 버킷에 media prefix 대상 lifecycle rule 추가:

```hcl
rule {
  id     = "media-to-glacier-ir"
  status = "Enabled"
  filter { prefix = "media/" }
  transition {
    days          = 7
    storage_class = "GLACIER_IR"
  }
}
```

**기존 Glacier IR 오브젝트**: lifecycle은 이미 Glacier IR인 오브젝트를 다시 전환하지 않음.
새 sync로 덮어쓰면 Standard로 올라가고, 7일 후 다시 Glacier IR로 전환됨.

#### 비용 효과

| | 요청 단가 (/1K) | 월 요청 비용 | 저장 비용 | 합계 |
|---|---|---|---|---|
| Before (GIR 직접) | $0.01 | ~$100 | ~$2.35 | **~$103** |
| After (Standard→lifecycle) | $0.0004 | ~$6 | ~$2.50 | **~$8.50** |

저장 비용 미세 증가 (7일간 Standard 단가), 요청 비용 94% 절감.

### 2. 라이브러리 통합

수동 작업 (Seafile 웹UI):

| Before | After | 작업 |
|--------|-------|------|
| 내 라이브러리 (18GB) | **내 라이브러리** — SSOT, 백업 대상 | 유지 |
| Google Drive (41GB) | ↑ 내 라이브러리로 이동 | 파일 이동 후 라이브러리 삭제 |
| LectureHub 과제 파일 | ↑ 내 라이브러리 `2026-1학기/`로 이동 | 파일 이동 |
| LectureHub 강의 영상 | **LectureHub** — 강의 영상 전용 | 유지 |
| My Library Template | 삭제 | 삭제 |

통합 후:

| 라이브러리 | 용도 | 예상 용량 | 백업 |
|-----------|------|----------|------|
| **내 라이브러리** | 개인 파일 전부 | ~59GB | Glacier IR (lifecycle) |
| **LectureHub** | 강의 영상만 | 향후 수백GB | **제외** |

### 3. LectureHub 백업 제외

Seafile 내부 스토리지 구조:

```
storage/{blocks,commits,fs}/<repo-uuid>/
```

LectureHub repo ID: `07c9cf44-8874-491b-9767-ef8783d407f5`

rclone sync에 exclude 추가:

```bash
rclone sync /seafile-data s3:seafile-backup-json-server/media/ \
  --exclude "storage/blocks/07c9cf44-8874-491b-9767-ef8783d407f5/**" \
  --exclude "storage/commits/07c9cf44-8874-491b-9767-ef8783d407f5/**" \
  --exclude "storage/fs/07c9cf44-8874-491b-9767-ef8783d407f5/**" \
  ...
```

**주의**: 라이브러리 삭제 후 재생성하면 UUID 변경됨 → exclude 업데이트 필요.

## 변경 대상 파일

### Terraform (aws/)

- `aws/s3.tf`: immich_backup, seafile_backup 버킷에 `media-to-glacier-ir` lifecycle rule 추가

### K8s manifests

- `k8s/immich/manifests/media-backup-cronjob.yaml`: rclone config에서 `storage_class = GLACIER_IR` 제거
- `k8s/seafile/manifests/media-backup-cronjob.yaml`: 위와 동일 + LectureHub UUID exclude 추가

## 작업 순서

1. **Terraform**: lifecycle rule 추가 → `terraform plan` → `terraform apply`
2. **K8s manifests**: rclone storage_class 제거 + exclude 추가
3. **CronJob suspend 해제**: 4개 모두 unsuspend
4. **검증**: 다음 날 CronJob 실행 로그 확인, S3 오브젝트가 Standard로 올라가는지 확인
5. **라이브러리 통합**: 웹UI에서 수동 수행 (별도 시점)
6. **통합 후**: seaf-gc 실행 (orphan 블록 정리), UUID 변경 있으면 exclude 업데이트

## 미래 최적화 (이번 스코프 밖)

- **옵션 2 (변경분만 sync)**: rclone `--max-age` 또는 Seafile 변경 이벤트 기반 sync
- **homelab-terraform IAM user Terraform import**: 현재 수동 관리, IaC로 전환
- **Factorio 백업 외부화**: 현재 클러스터 내 MinIO → 디스크 장애 시 같이 소실
