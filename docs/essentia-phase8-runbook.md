# Phase 8: Saemate → Essentia 앱 리브랜드 Runbook

Tracked in [essentia-edu/essentia#14](https://github.com/essentia-edu/essentia/issues/14).

이 문서는 **순서대로** 실행한다. 각 단계는 이전 단계 결과에 의존한다.

## 전제 조건

- [ ] VPN 연결 / 클러스터 도달 가능 (`kubectl get nodes` OK)
- [ ] `KUBECONFIG=~/.kube/config-json` 설정
- [ ] GCP OAuth redirect URI에 `https://api.essentia.json-server.win/auth/google/callback` 추가 완료 (유저 2026-04-19 완료)
- [ ] 구 URI `https://api.saemate.json-server.win/...` 삭제 완료 (유저 2026-04-19 완료, 결과적으로 saemate 로그인은 현재 깨진 상태 — 빠른 cutover 필요)

## 1. SealedSecret 재-seal (클러스터 접근 필요)

```bash
cd ~/homelab-worktrees/essentia-phase8-infra
KUBECONFIG=~/.kube/config-json ./scripts/reseal-essentia-secrets.sh
git diff k8s/essentia k8s/minio-tenants/overlays/essentia
```

세 파일이 실제 암호문으로 채워졌는지 확인:
- `k8s/essentia/api/sealed-secret.yaml`
- `k8s/essentia/ghcr-pull-sealed-secret.yaml`
- `k8s/minio-tenants/overlays/essentia/sealed-secret.yaml`

## 2. PR 머지

```bash
git add .
git commit
gh pr create
# PR 리뷰 후 머지
```

머지 직후:

- ArgoCD가 `essentia` + `essentia-minio` Application을 감지
- `essentia` namespace 생성 + Deployment + Ingress 배포 시도
- **주의**: DB는 비어있음 (PV 새로 생성). MinIO 버킷도 비어있음.
- `api.essentia.json-server.win` DNS 아직 없으므로 Ingress 접근 불가 (다음 단계에서 해결)

## 3. Cloudflare DNS apply (Terraform)

```bash
cd cloudflare
terraform plan
# 변경 검토 (4개 신규 record 추가: essentia, api.essentia, s3.essentia, minio.essentia)
terraform apply
```

**비용 영향 없음** (Cloudflare 무료 플랜 내). 유저 승인 후 실행.

## 4. Cert-manager 인증서 발급 대기

```bash
kubectl -n essentia get certificate
# ACME DNS01 challenge가 cert-manager로 자동 진행됨
# 2~5분 대기
```

인증서가 `Ready: True` 되면 HTTPS 가능.

## 5. 데이터 마이그레이션 (Postgres + MinIO)

### 5-1. Postgres pg_dump / pg_restore

```bash
# 5-1-a. saemate-api/web 스케일 다운 (쓰기 차단)
kubectl -n saemate scale deploy saemate-api --replicas=0
kubectl -n saemate scale deploy saemate-web --replicas=0

# 5-1-b. essentia-api/web 스케일 다운 (빈 DB에 덮어쓰기 방지)
kubectl -n essentia scale deploy essentia-api --replicas=0
kubectl -n essentia scale deploy essentia-web --replicas=0

# 5-1-c. saemate Postgres에서 dump
kubectl -n saemate exec deploy/saemate-postgres -- pg_dump -U saemate saemate > /tmp/saemate-dump.sql

# 5-1-d. essentia Postgres로 restore
# 먼저 DB 생성 확인
kubectl -n essentia exec deploy/essentia-postgres -- psql -U essentia -c "DROP DATABASE IF EXISTS essentia; CREATE DATABASE essentia;"
kubectl -n essentia exec -i deploy/essentia-postgres -- psql -U essentia essentia < /tmp/saemate-dump.sql

# 5-1-e. 용량/테이블 검증
kubectl -n essentia exec deploy/essentia-postgres -- psql -U essentia -d essentia -c "\dt"
```

### 5-2. MinIO mc mirror

```bash
# saemate MinIO → essentia MinIO
# mc alias 설정 (saemate-minio-api는 클러스터 내부에서 s3.saemate로 접근)
mc alias set src https://s3.saemate.json-server.win <saemate-access-key> <saemate-secret-key>
mc alias set dst https://s3.essentia.json-server.win <essentia-access-key> <essentia-secret-key>

# 버킷 복제 (saemate → essentia)
mc mirror --overwrite src/saemate dst/essentia
mc ls dst/essentia  # 검증
```

**키 획득**: `kubectl -n saemate get secret saemate-api-secrets -o jsonpath='{.data.MINIO_ACCESS_KEY}' | base64 -d` (동일하게 SECRET_KEY).

## 6. essentia 스택 기동

```bash
kubectl -n essentia scale deploy essentia-api --replicas=1
kubectl -n essentia scale deploy essentia-web --replicas=1
kubectl -n essentia get pods  # Ready 대기
```

## 7. 스모크 테스트 (유저)

브라우저:
1. https://essentia.json-server.win 접근
2. "Google로 로그인" 클릭 → OAuth flow 완료 (새 redirect URI 사용)
3. 강의 목록 페이지 로드 확인 (마이그레이션된 데이터)
4. 파일 업로드 1개 → MinIO 접근 확인
5. 강의 자료 다운로드 1개 → 이전 데이터 접근 확인

**실패 시**: saemate 스택이 여전히 살아있음 (scaled 0 but pods restartable). 긴급 롤백:
```bash
kubectl -n saemate scale deploy saemate-api --replicas=1
kubectl -n saemate scale deploy saemate-web --replicas=1
# DNS는 essentia.* 유지해도 되지만 API URL이 saemate용이라 로그인 안 됨 — OAuth URI 복구 필요
```

## 8. Cutover 정리 (성공 시)

```bash
# ArgoCD app 제거
kubectl -n argocd delete application saemate saemate-minio
# 또는: k8s/argocd/applications/apps/saemate.yaml, saemate-minio.yaml 파일 삭제 → PR 머지

# 구 ns 삭제 (데이터 영구 삭제, 최소 1주 유예 권장)
# kubectl delete ns saemate
```

## 9. DNS 레거시 정리 (1-2주 뒤)

```bash
# cloudflare/dns.tf 에서 saemate_* 4개 리소스 블록 제거
cd cloudflare
terraform plan  # destroy 4개 확인
terraform apply
```

## 10. 워크트리 경로 이전 (Claude 세션 종료 후)

```bash
mv ~/saemate-worktrees/saemate ~/essentia-worktrees/essentia
cd ~/essentia-worktrees/essentia/main
git worktree repair
```

`.code-workspace` 경로도 동시 갱신.

## 10-1. GHCR 구 이미지 정리 (유저 UI)

https://github.com/orgs/essentia-edu/packages → `saemate/api`, `saemate/web` 삭제 (트래픽 0 확인 후).

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| `pg_dump` 중 saemate에 쓰기 발생 | scale=0 선행 (5-1-a) |
| `mc mirror` 중 saemate에 쓰기 발생 | 위와 동일 |
| 구 saemate 스택 데이터 유실 (roll-forward 직전) | saemate ns 삭제는 cutover 성공 + 1주 유예 후 |
| Sealed-secrets controller 내부 private key 로테이션 | cert.pem 최신 확인 (`kubeseal --fetch-cert` 필요 시) |
| Google OAuth redirect URI 실수 | 신규 URI `api.essentia.*` 이미 추가됨 (유저 완료) |
