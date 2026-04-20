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

## 자동 초기화 · 마이그레이션

`migrate-job.yaml` (ArgoCD PostSync hook)이 sync마다 실행:

1. `prisma migrate deploy` — 미적용 migration 전부 apply (hypertable SQL 포함, Loop#8)
2. `pnpm db:seed` — roles · domains · habits · devices upsert

모든 단계 idempotent. sync 반복 실행 안전.

### 수동 재실행

```bash
kubectl -n loop delete job loop-migrate
kubectl -n argocd patch app loop --type merge -p '{"operation":{"sync":{}}}'
# 또는 템플릿에서 새 Job 생성
kubectl -n loop create job --from=job/loop-migrate loop-migrate-$(date +%s)
```

### Job 로그 확인

```bash
kubectl -n loop logs job/loop-migrate --all-containers --tail=200
```

## 스키마 변경 시

1. Loop 레포 워크트리에서 `prisma/schema.prisma` 수정
2. `pnpm --filter @loop/db exec prisma migrate dev --name <desc>` → migration 생성·커밋·Loop PR 셀프 머지
3. GitHub Actions build-push → GHCR에 새 이미지
4. ArgoCD Image Updater가 digest 감지 → Deployment + Job 둘 다 새 이미지
5. Job 실행 → `prisma migrate deploy` 가 신규 migration 자동 apply

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
