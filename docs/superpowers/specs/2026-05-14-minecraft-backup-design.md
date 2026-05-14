# Minecraft 월드 백업 — itzg/mc-backup + restic → S3

> Issue: [manamana32321/homelab#205](https://github.com/manamana32321/homelab/issues/205)
> Date: 2026-05-14
> 도구 선정 근거: [k8s/minecraft/README.md](../../../k8s/minecraft/README.md) (PR [#210](https://github.com/manamana32321/homelab/pull/210))

## 배경

마인크래프트 서버는 현재 백업이 없다. 월드 데이터는 `minecraft-datadir` PVC에만 존재 — 노드/디스크 장애 시 소실. 월드는 현재 비어있다시피 하지만 친구들과 플레이가 시작되면 데이터가 쌓인다. 데이터 보호는 다른 운영 도구 도입의 안전망이므로 가장 먼저 진행한다.

## 설계 개요

**itzg/mc-backup 사이드카 컨테이너 + restic 백엔드 → AWS S3.**

- **사이드카**: 마크 Deployment에 컨테이너 추가 (`values.yaml`의 `mcbackup.*`) — 순수 선언적, itzg 차트 5.0.0 내장
- **restic**: 증분 + 중복제거 + 암호화 스냅샷. 작은 월드도 효율적, point-in-time 복원
- **일관성**: mc-backup이 RCON으로 `save-off`/`save-on`을 자동 조율 → 쓰기 도중 스냅샷 방지
- **S3**: 기존 백업 인프라(`{app}-backup-json-server` 버킷 + 전용 IAM 사용자 패턴)에 합류

```
마크 서버 컨테이너 ──┐
                    ├─ 같은 Pod, /data PVC 공유
mc-backup 사이드카 ──┘
   │  RCON save-off → restic backup /data/world → save-on
   ▼
AWS S3: minecraft-backup-json-server (restic 저장소)
```

## 구성 요소

### 1. S3 버킷 + IAM (Terraform: `aws/s3.tf`, `aws/iam.tf`)

immich/seafile/health와 동일 패턴:

- **버킷**: `minecraft-backup-json-server` (ap-northeast-2) + public access block
- **IAM 사용자**: `minecraft-backup` + 정책(`s3:PutObject/GetObject/ListBucket/DeleteObject`) + access key
  - restic은 `prune` 시 객체 삭제 → `DeleteObject` 필수 (rclone 백업과 동일 액션 셋)
- access key는 Terraform output(sensitive) → 수동으로 SealedSecret에 sealing

### 2. SealedSecret (`k8s/minecraft/`)

`minecraft-mcbackup` Secret — `mcbackup.envFrom`로 사이드카에 주입:

| 키 | 용도 |
|---|---|
| `RESTIC_PASSWORD` | restic 저장소 암호화 비밀번호 |
| `RESTIC_REPOSITORY` | `s3:s3.ap-northeast-2.amazonaws.com/minecraft-backup-json-server` |
| `AWS_ACCESS_KEY_ID` | `minecraft-backup` IAM 사용자 키 |
| `AWS_SECRET_ACCESS_KEY` | 〃 |

> ⚠️ `RESTIC_PASSWORD`를 잃으면 백업 전체가 복구 불능. SealedSecret은 git에 암호화되어 있지만, 평문 비밀번호는 별도 안전한 곳(비밀번호 관리자 등)에도 보관할 것.

### 3. `values.yaml` — `mcbackup.*`

```yaml
mcbackup:
  enabled: true
  backupMethod: restic
  resticHostname: minecraft-homelab          # 고정 필수 (아래 리스크 참조)
  backupInterval: 24h
  pauseIfNoPlayers: "true"                   # 결정 2
  pruneResticRetention: "--keep-daily 7 --keep-weekly 4"  # 결정 1
  excludes: "*.jar,cache,logs"               # 차트 기본값 — jar/로그/캐시 제외
  envFrom:
    - secretRef:
        name: minecraft-mcbackup
  resources:
    requests: { memory: 512Mi, cpu: 250m }
    limits:   { memory: 512Mi, cpu: 500m }
```

`resticRepository`는 SealedSecret의 `RESTIC_REPOSITORY` 환경변수로 주입(values에 평문으로 안 박음).

## 결정 사항 (확정 2026-05-14)

### 결정 1 — 보존 정책: `--keep-daily 7 --keep-weekly 4`

약 1개월 히스토리. 친구 몇 명 규모엔 충분하고 스냅샷 수가 적어 관리가 단순. 차트 기본값(`--keep-yearly 75`까지)은 친구 서버에 과해서 채택 안 함.

### 결정 2 — idle 시 백업: `pauseIfNoPlayers: "true"`

아무도 안 들어온 날은 월드 변화가 없으니 스냅샷 생략. 스냅샷 개수가 retention 계산에 영향을 주므로 불필요한 스냅샷을 만들지 않음.

### 결정 3 — S3 스토리지 클래스: Standard

백업 데이터가 작다(월드 dedup + jar/로그 제외 → 수십~수백 MB). IT의 오브젝트당 모니터링 비용은 작은 데이터셋엔 무의미하고, restic의 prune/check가 pack 파일을 읽으므로 Standard가 단순. 비용 차이 ~$0.01/월 수준. → `aws/s3.tf`에 lifecycle rule 없음(restic이 자체 retention 관리), 기본 Standard.

> **사이드카 스케줄링 한계**: itzg/mc-backup은 cron이 아닌 interval 기반 → "새벽 KST 정각" 정밀 타격 불가. pod 시작 시점 기준 24h 주기로 드리프트. 정밀 cron이 필요하면 사이드카 대신 별도 CronJob을 짜야 하는데 RCON 조율을 직접 구현해야 함(tech-scout가 사이드카를 권고한 이유). → interval 모델 수용.

## 구현 단계 (#205 acceptance)

1. [ ] `aws/s3.tf` — `minecraft-backup-json-server` 버킷 + public access block (+ 결정 3에 따라 lifecycle)
2. [ ] `aws/iam.tf` — `minecraft-backup` IAM 사용자/정책/attachment/access key
3. [ ] `terraform plan` 확인 → 사용자 승인 → `apply`
4. [ ] access key + restic 비밀번호 → `minecraft-mcbackup` SealedSecret 생성 (`KUBECONFIG=~/.kube/config-<homelab>` 명시)
5. [ ] `k8s/minecraft/values.yaml` — `mcbackup.*` 블록 추가
6. [ ] PR → 머지 → ArgoCD sync 검증 (사이드카 컨테이너 Ready, 첫 백업 로그 확인)
7. [ ] **복원 리허설** — 아래 절차로 1회 실제 복원 테스트
8. [ ] `k8s/minecraft/README.md` 현재 상태 표 갱신

## 복원 절차

restic 복원은 mc-backup 컨테이너 안에서:

```bash
kubectl exec -n minecraft deploy/minecraft -c mc-backup -- \
  restic snapshots                              # 스냅샷 목록
kubectl exec -n minecraft deploy/minecraft -c mc-backup -- \
  restic restore <snapshot-id> --target /data   # 특정 시점 복원
```

복원 시 마크 서버는 `save-off` 상태여야 안전 → 실제 복원은 deploy scale 0 후 진행.

## 검증

- 사이드카 `mc-backup` 컨테이너 Ready, RestartCount 안정
- 첫 백업 후 `restic snapshots`에 스냅샷 1개 이상
- S3 버킷에 restic 저장소 구조(`config`, `data/`, `snapshots/`, `index/`) 생성됨
- 복원 리허설 성공 (스냅샷 → 임시 타겟 복원 → 월드 파일 존재 확인)

## 리스크 / 주의

- **`resticHostname` 고정 필수**: pod 재생성마다 컨테이너 hostname이 바뀌면 restic이 이전 호스트의 스냅샷을 prune 못 함 → 스냅샷 무한 누적. 반드시 고정값 지정.
- **`RESTIC_PASSWORD` 분실 = 백업 전체 복구 불능**: SealedSecret(git, 암호화) 외에 평문도 별도 보관.
- **사이드카 메모리**: mc-backup 컨테이너가 512Mi 요청 — 마크 서버 Pod 전체 리소스가 그만큼 늘어남. 노드(json-server-1) 여유 확인.
- **첫 백업 타이밍**: `initialDelay: 2m` 후 시작 → interval 24h. 정밀 스케줄 불가(위 참조).
- **stateful 함정 재확인**: 복원은 반드시 `save-off`/scale-0 상태에서. 실행 중 복원하면 월드 손상.
