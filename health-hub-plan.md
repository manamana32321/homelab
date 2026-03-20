# Health Hub — 건강 데이터 중앙 관리 시스템

> 작성일: 2026-03-20
> 목표: Samsung Health 데이터를 자동 수집하여 홈랩에서 시각화 및 분석
> 사용자: 손장수 (1인, Galaxy Watch4 + Galaxy Z Flip 5)
> 도메인: health.json-server.win

---

## 1. 프로젝트 개요

### 핵심 요구사항

- 폰에서 서버로 데이터 자동 전송 (Tasker, 사용자 조작 제로)
- 수면, 걸음수, 심박수, 운동, 체중, 산소포화도, 체성분, 음식 수집
- K3s 홈랩 클러스터에 배포 (ArgoCD GitOps)
- 추후 MCP/OpenClaw 연동 고려

### 범위 제외 (Phase 1)

- 마음챙김 데이터 (Health Connect 미지원)
- 스트레스 지수 (Samsung 독점, Health Connect 미지원)
- CGM (추후 Phase 3)
- 커스텀 프론트엔드 (Phase 2, 초기에는 Grafana)

---

## 2. 아키텍처

```
┌─ Phone (Galaxy Z Flip 5) ──────────────────────────┐
│                                                     │
│  Galaxy Watch4                                      │
│      ↓ BLE                                          │
│  Samsung Health                                     │
│      ↓ auto-sync (Android 13+)                      │
│  Health Connect                                     │
│      ↓ read (15분 간격)                              │
│  Tasker + Health Connect plugin                     │
│      ↓ HTTP POST                                    │
│  health-hub API                                     │
│                                                     │
└─────────────────────────────────────────────────────┘
                        ↓ HTTPS
┌─ K3s Cluster ───────────────────────────────────────┐
│                                                     │
│  health-hub (Go, 1 replica)                         │
│    ├─ POST /api/v1/ingest   ← Tasker 데이터 수신    │
│    ├─ GET  /api/v1/metrics  ← Grafana/MCP 조회      │
│    └─ Bearer token 인증                              │
│         ↓                                           │
│  TimescaleDB (PostgreSQL + extension)               │
│    ├─ hypertable: health_metrics                    │
│    ├─ table: exercise_sessions                      │
│    ├─ table: sleep_sessions                         │
│    ├─ table: nutrition_records                      │
│    └─ longhorn-ssd PVC (5Gi)                        │
│         ↓                                           │
│  Grafana (기존 스택)                                 │
│    └─ TimescaleDB datasource → 건강 대시보드         │
│                                                     │
└─────────────────────────────────────────────────────┘
```

---

## 3. 수집 데이터 상세

### Health Connect → Tasker → 서버

| 데이터 | Health Connect 타입 | 수집 간격 | 단위 |
|--------|-------------------|----------|------|
| 걸음수 | `StepsRecord` | 15분 | steps |
| 심박수 | `HeartRateRecord` | 15분 | bpm |
| 수면 | `SleepSessionRecord` + stages | 1시간 | 분 (단계별) |
| 운동 | `ExerciseSessionRecord` | 15분 | 종류, 시간, kcal |
| 체중 | `WeightRecord` | 1시간 | kg |
| 산소포화도 | `OxygenSaturationRecord` | 15분 | % |
| 체성분 | `BodyFatRecord`, `LeanBodyMassRecord` | 1시간 | %, kg |
| 음식 | `NutritionRecord` | 1시간 | kcal, 영양소 |
| 수분 섭취 | `HydrationRecord` | 1시간 | mL |
| 거리 | `DistanceRecord` | 15분 | m |
| 칼로리 | `TotalCaloriesBurnedRecord` | 15분 | kcal |

---

## 4. 기술 스택

### Backend: Go

```
health-hub/
├── cmd/server/main.go       # 엔트리포인트
├── internal/
│   ├── api/                  # HTTP 핸들러
│   │   ├── ingest.go         # POST /api/v1/ingest
│   │   ├── query.go          # GET /api/v1/metrics
│   │   └── middleware.go     # 인증, 로깅
│   ├── db/                   # TimescaleDB 접근
│   │   ├── migrations/       # SQL 마이그레이션
│   │   └── repository.go
│   └── model/                # 데이터 모델
├── Dockerfile
├── go.mod
└── go.sum
```

주요 의존성:
- `net/http` (stdlib, 프레임워크 불필요)
- `jackc/pgx/v5` (PostgreSQL 드라이버)
- `golang-migrate/migrate` (DB 마이그레이션)
- `slog` (stdlib 로깅)

### Database: TimescaleDB

단일 인스턴스. PostgreSQL 확장이므로 메타데이터 테이블도 같은 DB에 공존.

```sql
-- 시계열 메트릭 (걸음, 심박, 산소포화도, 칼로리, 거리, 수분)
CREATE TABLE health_metrics (
    time        TIMESTAMPTZ NOT NULL,
    metric_type TEXT        NOT NULL,  -- 'steps', 'heart_rate', 'spo2', ...
    value       DOUBLE PRECISION NOT NULL,
    unit        TEXT        NOT NULL,
    source      TEXT DEFAULT 'samsung_health',
    metadata    JSONB
);
SELECT create_hypertable('health_metrics', 'time');
CREATE INDEX idx_metrics_type_time ON health_metrics (metric_type, time DESC);

-- 수면 세션
CREATE TABLE sleep_sessions (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    start_time  TIMESTAMPTZ NOT NULL,
    end_time    TIMESTAMPTZ NOT NULL,
    duration_m  INT NOT NULL,
    stages      JSONB NOT NULL  -- [{stage: "deep", start: ..., end: ..., duration_m: ...}]
);

-- 운동 세션
CREATE TABLE exercise_sessions (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    exercise_type   TEXT NOT NULL,       -- 'running', 'walking', 'cycling', ...
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ NOT NULL,
    duration_m      INT NOT NULL,
    calories_kcal   DOUBLE PRECISION,
    distance_m      DOUBLE PRECISION,
    metadata        JSONB
);

-- 음식/영양
CREATE TABLE nutrition_records (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    time        TIMESTAMPTZ NOT NULL,
    meal_type   TEXT,                     -- 'breakfast', 'lunch', 'dinner', 'snack'
    calories    DOUBLE PRECISION,
    protein_g   DOUBLE PRECISION,
    fat_g       DOUBLE PRECISION,
    carbs_g     DOUBLE PRECISION,
    metadata    JSONB                     -- 기타 영양소
);

-- 체성분 (체중 포함)
CREATE TABLE body_measurements (
    time            TIMESTAMPTZ NOT NULL,
    weight_kg       DOUBLE PRECISION,
    body_fat_pct    DOUBLE PRECISION,
    lean_mass_kg    DOUBLE PRECISION
);
SELECT create_hypertable('body_measurements', 'time');
```

### Visualization: Grafana

기존 Grafana (grafana.json-server.win)에 TimescaleDB datasource 추가.
대시보드는 JSON provisioning으로 GitOps 관리.

### Infrastructure

```yaml
# homelab 레포 내 위치
k8s/
├── argocd/applications/apps/
│   └── health-hub.yaml              # ArgoCD Application
└── health-hub/
    ├── manifests/
    │   ├── namespace.yaml
    │   ├── timescaledb-statefulset.yaml
    │   ├── health-hub-deployment.yaml
    │   ├── health-hub-service.yaml
    │   ├── ingress.yaml
    │   ├── sealed-secret.yaml        # API token, DB password
    │   └── backup-cronjob.yaml       # S3 백업
    └── grafana/
        └── health-dashboard.json     # Grafana provisioning
```

---

## 5. API 설계

### 인증

Bearer token (SealedSecret으로 관리). 1인 사용이므로 OAuth/JWT 불필요.

### 엔드포인트

```
POST /api/v1/ingest
  Body: { metrics: [...], sleep_sessions: [...], exercises: [...], ... }
  → Tasker가 보내는 벌크 데이터 수신
  → 중복 제거 (timestamp + metric_type 기준)

GET /api/v1/metrics?type=steps&from=2026-03-01&to=2026-03-20&interval=1h
  → 시계열 조회 (집계 간격 지정 가능)

GET /api/v1/sleep?from=2026-03-01&to=2026-03-20
  → 수면 세션 조회

GET /api/v1/exercises?from=2026-03-01&to=2026-03-20
  → 운동 세션 조회

GET /api/v1/nutrition?from=2026-03-01&to=2026-03-20
  → 영양 기록 조회

GET /api/v1/body?from=2026-03-01&to=2026-03-20
  → 체성분/체중 조회

GET /api/v1/summary?date=2026-03-20
  → 일별 요약 (전 항목 통합)

GET /healthz
  → 헬스체크 (DB 연결 확인)
```

### Ingest 페이로드 예시

```json
{
  "timestamp": "2026-03-20T14:30:00+09:00",
  "metrics": [
    { "type": "steps", "value": 342, "unit": "count", "time": "2026-03-20T14:15:00+09:00" },
    { "type": "heart_rate", "value": 72, "unit": "bpm", "time": "2026-03-20T14:20:00+09:00" },
    { "type": "spo2", "value": 97, "unit": "percent", "time": "2026-03-20T14:25:00+09:00" }
  ],
  "sleep_sessions": [],
  "exercises": [],
  "nutrition": [],
  "body": []
}
```

---

## 6. Tasker 설정 (사용자 1회 설정 후 자동)

### 필요 앱

1. **Tasker** (유료, ~$4)
2. **Health Connect Tasker Plugin** (무료)
3. **Health Connect** (Samsung Health 연동 확인)

### 프로파일 구성

```
Profile: Health Sync (15분 간격)
  Trigger: Time → Every 15 Minutes
  Task:
    1. Health Connect Plugin → Read Steps (last 15min)
    2. Health Connect Plugin → Read Heart Rate (last 15min)
    3. Health Connect Plugin → Read SpO2 (last 15min)
    4. Health Connect Plugin → Read Calories (last 15min)
    5. Health Connect Plugin → Read Distance (last 15min)
    6. Variable Set → Build JSON payload
    7. HTTP Request → POST https://health.json-server.win/api/v1/ingest
       Headers: Authorization: Bearer %TOKEN%
       Body: %json_payload%

Profile: Health Sync Hourly (1시간 간격)
  Trigger: Time → Every 1 Hour
  Task:
    1. Health Connect Plugin → Read Sleep Sessions (last 2h, 겹침 방지)
    2. Health Connect Plugin → Read Weight (last 2h)
    3. Health Connect Plugin → Read Body Fat (last 2h)
    4. Health Connect Plugin → Read Nutrition (last 2h)
    5. Health Connect Plugin → Read Hydration (last 2h)
    6. Variable Set → Build JSON payload
    7. HTTP Request → POST https://health.json-server.win/api/v1/ingest
```

### 배터리 최적화

- Tasker → 배터리 최적화 제외 설정 필수
- Health Connect → 배터리 최적화 제외
- Samsung Health → 백그라운드 활동 허용

---

## 7. 보안

| 항목 | 구현 |
|------|------|
| API 인증 | Bearer token (SealedSecret) |
| HTTPS | cert-manager DNS01 (Cloudflare) |
| DB 접근 | ClusterIP (내부 전용), password (SealedSecret) |
| Tasker → 서버 | HTTPS + Bearer token |
| 백업 암호화 | S3 서버사이드 암호화 |

---

## 8. 백업

기존 Immich/Seafile 패턴과 동일:

- **S3 버킷**: `health-backup-json-server` (ap-northeast-2)
- **IAM user**: `health-backup`
- **DB 백업**: daily 17:00 UTC (02:00 KST), pg_dump → gzip → S3 Standard
- **보관**: 30일 lifecycle rule
- **SealedSecret**: `health-backup-credentials`
- **CronJob**: `k8s/health-hub/manifests/backup-cronjob.yaml`

---

## 9. 모니터링

기존 OTel 스택 활용:
- Go 앱 → stdout 로그 → Loki (기존 수집 파이프라인)
- `/healthz` → Blackbox Exporter (기존 alert-rules.yaml에 추가)
- TimescaleDB 메트릭 → Prometheus (pg_exporter 또는 쿼리 기반)

---

## 10. 구현 단계

### Phase 1: 서버 인프라 ✅ 구현 완료 (2026-03-20)

| Step | 작업 | 상태 |
|------|------|------|
| 1-1 | Go 프로젝트 scaffold + Dockerfile | ✅ `health-hub/` |
| 1-2 | TimescaleDB K8s manifests | ✅ `k8s/health-hub/manifests/timescaledb-statefulset.yaml` |
| 1-3 | DB 마이그레이션 SQL (embedded) | ✅ `health-hub/internal/db/migrations/001_init.sql` |
| 1-4 | Ingest API 구현 + 중복 제거 | ✅ 17 tests passing |
| 1-5 | Query API 구현 (metrics/sleep/exercises/nutrition/body/summary) | ✅ |
| 1-6 | health-hub K8s manifests | ✅ Deployment, Service, Ingress, SealedSecret |
| 1-7 | ArgoCD Application | ✅ `k8s/argocd/applications/apps/health-hub.yaml` |
| 1-8 | Grafana 대시보드 + datasource ConfigMap | ✅ sidecar provisioning |
| 1-9 | S3 백업 CronJob + Terraform (bucket, IAM) | ✅ `aws/s3.tf`, `aws/iam.tf` |
| 1-10 | Cloudflare DNS | ✅ `cloudflare/dns.tf` (proxied=true) |
| 1-11 | GitHub Actions CI (test → build → push) | ✅ `.github/workflows/health-hub-image.yaml` |

**배포 전 수동 작업:**
1. `kubeseal`로 `sealed-secret.yaml` 생성 (DB_PASSWORD, API_TOKEN)
2. `cd aws && terraform apply` (S3 버킷 + IAM 생성)
3. `kubeseal`로 `sealed-secret-backup.yaml` 생성 (AWS credentials)
4. `grafana-datasource.yaml`의 DB 비밀번호 업데이트
5. `cd cloudflare && terraform apply` (DNS 레코드)
6. Git push → GitHub Actions → ArgoCD 자동 배포

### Phase 1 보류: Tasker 설정 (사용자 수동, 최후순위)

| Step | 작업 |
|------|------|
| T-1 | Health Connect 활성화 + Samsung Health 동기화 확인 |
| T-2 | Tasker 설치 + Health Connect 플러그인 설치 |
| T-3 | Tasker 프로파일 설정 (15분 + 1시간) |
| T-4 | 배터리 최적화 제외 설정 |
| T-5 | 동작 확인 (수동 트리거 → 서버 로그 확인) |

### Phase 2: 커스텀 대시보드 + AI

- [ ] Next.js 대시보드 (health.json-server.win 또는 별도 서브도메인)
- [ ] MCP Server (health-hub API 래핑)
- [ ] OpenClaw 플러그인 ("오늘 운동 얼마나 했어?", "이번 주 수면 패턴 분석해줘")
- [ ] 상관관계 분석 (수면-운동, 체중-칼로리)
- [ ] 이상치 탐지 + Telegram 알림

### Phase 3: CGM + 고급

- [ ] LibreLink/LibreView API 연동
- [ ] AI 건강 코칭 (래미/자비스 통합)
- [ ] HRV 기반 스트레스 추정
- [ ] PDF 보고서 자동 생성

---

## 11. 의사결정 기록

| 결정 | 이유 |
|------|------|
| Google Fit API 사용 안 함 | 2025-06 종료됨 |
| Health Connect + Tasker | 서버사이드 API 없는 Samsung 생태계에서 유일한 완전 자동화 경로 |
| Go (NestJS 대신) | 단일 바이너리, 의존성 제로, K3s 리소스 절약 |
| TimescaleDB 단일 인스턴스 | PG 확장이므로 분리 불필요, 메타데이터+시계열 공존 |
| Grafana (Next.js 대신 Phase 1) | 이미 운영 중, 시계열 시각화 네이티브 지원, 개발 비용 제로 |
| MinIO 별도 추가 안 함 | Phase 1에 오브젝트 스토리지 필요 없음 |
| Bearer token (JWT 대신) | 1인 사용, 복잡한 인증 불필요 |
| ArgoCD GitOps | 홈랩 표준 배포 패턴 |
| S3 백업 (MinIO 대신) | 기존 Immich/Seafile 패턴 재활용 |
| 마음챙김/스트레스 스킵 | Health Connect 미지원 또는 Samsung 독점 |
