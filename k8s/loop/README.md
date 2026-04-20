# Loop

중앙 Habit Hub. 소스: [manamana32321/loop](https://github.com/manamana32321/loop)

## 구조

```
k8s/loop/
├── namespace.yaml
├── ghcr-pull-sealed-secret.yaml      # private GHCR 이미지 pull
├── kustomization.yaml
├── postgres/                         # TimescaleDB (Deployment + PVC + Service)
├── api/                              # NestJS — Deployment + Service + ConfigMap + SealedSecret
└── web/                              # Next.js — Deployment + Service + Ingress(habits.json-server.win + Authentik)
```

## 최초 배포 절차 (수동 단계)

ArgoCD가 리소스를 배포하면 Postgres만 올라오고 스키마·hypertable·seed는 비어 있음.
API 이미지에 Prisma CLI/tsx가 devDependency라서 자동 초기화 Job은 별도 PR로 분리 예정.
첫 배포 시 로컬에서 한 번 다음을 실행:

```bash
# 1. 로컬에서 클러스터로 port-forward
kubectl -n loop port-forward svc/loop-postgres 5433:5432 &

# 2. Loop 레포 루트에서
cd ~/loop-worktrees/main

# 3. DB URL 지정 (패스워드는 loop-api-secrets SealedSecret에서 조회)
PG_PASS=$(kubectl -n loop get secret loop-api-secrets -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -d)
export DATABASE_URL="postgresql://loop:${PG_PASS}@localhost:5433/loop?schema=public"

# 4. Prisma migration
pnpm --filter @loop/db exec prisma migrate deploy

# 5. TimescaleDB hypertable
psql "$DATABASE_URL" -f packages/db/prisma/post-migration/01_timescale_hypertable.sql

# 6. Seed
pnpm db:seed

# 7. 정리
kill %1
unset DATABASE_URL
```

## 스키마 변경 시

1. Loop 레포에서 `prisma/schema.prisma` 수정 → `prisma migrate dev --name <desc>` 로 migration 생성·커밋
2. CI/CD가 api 이미지 재빌드
3. ArgoCD Image Updater가 새 digest 감지 → 배포
4. (현재는 수동) 위 port-forward + `prisma migrate deploy` 로 DB 스키마 적용

## TLS / 접근

- Host: `habits.json-server.win` (Cloudflare proxied, Universal SSL)
- Authentik forward auth 필수 (개인 데이터)
- API는 cluster-internal만. `loop-api.loop.svc.cluster.local:3001`
- Web이 server component fetch로 API 호출. API 외부 노출 없음.

## 향후 과제

- [ ] migrate Job 자동화 (별도 migrate 이미지 또는 api 이미지에 prisma CLI 포함)
- [ ] Telegram bot 입력 채널 → habit_events 자동 로깅
- [ ] Grafana dashboard for habit heatmap/streak
- [ ] Health Hub 연동 (Samsung Health 데이터 → habit_events)
- [ ] 관계 CRM UI 확장
