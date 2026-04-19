# Health Hub 앱 레포 분리 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** homelab 레포 안의 `health-hub/`(Go 백엔드)와 `health-hub-dashboard/`(Next.js) 앱 코드를 `manamana32321/health-hub` private monorepo로 히스토리 보존하며 분리. k8s 매니페스트·ArgoCD Application은 homelab에 유지.

**Architecture:**
- 앱 코드만 분리 → 새 레포 monorepo 구조: `backend/` + `dashboard/`
- `git filter-repo --path-rename` 단일 명령으로 필터링 + 리네임 원자적 수행
- GHCR 이미지 이름 유지 (`ghcr.io/manamana32321/health-hub[-dashboard]`), 패키지 visibility public 유지 → K3s/ArgoCD 변경 불필요
- Image Updater `write-back-method: argocd` 사용 중이라 소스 레포 경로 변경 무영향

**Tech Stack:** Go 1.24, Next.js 16 (pnpm), Docker, GitHub Actions, ArgoCD Image Updater, git filter-repo

**Rollback Plan:** 새 레포는 언제든 삭제 가능. homelab cleanup PR은 머지 전까지 revert 가능. filter-repo 작업은 별도 bare clone에서 수행하므로 원본 homelab git 상태 영향 없음.

---

## File Map

**새 레포 `manamana32321/health-hub` 구조 (최종):**

```
health-hub/
├── .github/workflows/
│   ├── backend-image.yaml        # paths: backend/**
│   └── dashboard-image.yaml      # paths: dashboard/**
├── backend/                      # ex-health-hub/
│   ├── Dockerfile
│   ├── cmd/{server,mcp}/main.go
│   ├── internal/{api,db,model}/
│   ├── go.mod                    # module github.com/manamana32321/health-hub/backend
│   └── go.sum
├── dashboard/                    # ex-health-hub-dashboard/
│   ├── Dockerfile
│   ├── AGENTS.md
│   ├── CLAUDE.md -> AGENTS.md
│   ├── app/, components/, lib/, public/
│   ├── package.json, pnpm-lock.yaml, pnpm-workspace.yaml
│   └── ...
├── CLAUDE.md                     # 앱 전용 컨텍스트
├── README.md                     # 프로젝트 소개 + homelab 레포 크로스링크
└── .gitignore                    # Go + Node 통합
```

**homelab 레포 변경:**

- Delete: `health-hub/`, `health-hub-dashboard/`, `.github/workflows/health-hub-image.yaml`, `.github/workflows/health-hub-dashboard-image.yaml`
- Keep unchanged: `k8s/health-hub/`, `k8s/argocd/applications/apps/health-hub.yaml`, `k8s/observability/grafana/health-hub-dashboard.yaml`
- Update: `CLAUDE.md` — health-hub 관련 언급이 있으면 새 레포 링크로 교체

---

## Task 1: 사전 검증 및 워킹 카피 준비

**Files:**
- Create: `/tmp/health-hub-split/` (임시 작업 공간)

- [ ] **Step 1: 현재 main 커밋 해시 기록**

```bash
cd /home/json/homelab-worktrees/main
git log -1 --format="%H %s" > /tmp/health-hub-split/source-commit.txt
cat /tmp/health-hub-split/source-commit.txt
```

- [ ] **Step 2: git-filter-repo 설치 확인**

```bash
git filter-repo --version || pip install --user git-filter-repo
git filter-repo --version
```

Expected: `git filter-repo <version>` 출력. 없으면 apt/pip으로 설치.

- [ ] **Step 3: 분리 대상 파일 인벤토리**

```bash
cd /home/json/homelab-worktrees/main
find health-hub health-hub-dashboard -type f | wc -l
find health-hub health-hub-dashboard -type f > /tmp/health-hub-split/inventory-before.txt
```

기록: 총 파일 수. filter-repo 후 새 레포의 파일 수와 대조용.

- [ ] **Step 4: GitHub CLI 인증 확인**

```bash
gh auth status
```

Expected: `Logged in to github.com account manamana32321`. 없으면 `gh auth login`.

---

## Task 2: filter-repo로 새 레포 히스토리 생성

**Files:**
- Create: `/tmp/health-hub-split/health-hub.git` (bare clone)
- Create: `/tmp/health-hub-split/health-hub-working/` (filter 적용된 워킹 카피)

- [ ] **Step 1: homelab 레포 bare clone**

```bash
cd /tmp/health-hub-split
git clone --bare https://github.com/manamana32321/homelab.git health-hub.git
cd health-hub.git
git log --oneline | head -3
```

Expected: 최근 커밋 3개 출력.

- [ ] **Step 2: filter-repo로 경로 필터링 + 리네임**

```bash
cd /tmp/health-hub-split/health-hub.git
git filter-repo \
  --path health-hub/ \
  --path health-hub-dashboard/ \
  --path-rename health-hub/:backend/ \
  --path-rename health-hub-dashboard/:dashboard/
```

Expected: "Parsed N commits" 메시지. `refs/heads/main`만 남고 나머지 브랜치는 삭제됨 (origin remote 제거됨).

- [ ] **Step 3: filter 결과 검증 — non-bare 체크아웃**

```bash
cd /tmp/health-hub-split
git clone health-hub.git health-hub-working
cd health-hub-working
ls
# Expected: backend/  dashboard/
find backend dashboard -type f | wc -l
# Expected: Task 1 Step 3의 파일 수와 동일
git log --oneline | wc -l
# Expected: health-hub/, health-hub-dashboard/ 경로가 포함된 커밋만 (전체 homelab 커밋보다 훨씬 적음)
```

- [ ] **Step 4: filter 결과 세부 확인**

```bash
cd /tmp/health-hub-split/health-hub-working
git log --oneline --all | head -20
```

Expected: health-hub 관련 커밋 메시지만 보임 (`feat(health-hub): ...`, `fix(health-hub): ...` 등). `feat(windmill)` 같은 무관 커밋 없음.

- [ ] **Step 5: 커밋**

이 단계는 commit하지 않음 (현재 워크트리 변경 없음). `/tmp/health-hub-split/`는 임시 공간.

---

## Task 3: Go 모듈 경로 재작성

**Files:**
- Modify: `/tmp/health-hub-split/health-hub-working/backend/go.mod`
- Modify: `/tmp/health-hub-split/health-hub-working/backend/cmd/server/main.go`
- Modify: `/tmp/health-hub-split/health-hub-working/backend/cmd/mcp/main.go`
- Modify: `/tmp/health-hub-split/health-hub-working/backend/internal/api/*.go`
- Modify: `/tmp/health-hub-split/health-hub-working/backend/internal/db/*.go`

- [ ] **Step 1: 모듈 경로 일괄 치환**

```bash
cd /tmp/health-hub-split/health-hub-working
grep -rln "github.com/manamana32321/homelab/health-hub" backend/ | \
  xargs sed -i 's|github.com/manamana32321/homelab/health-hub|github.com/manamana32321/health-hub/backend|g'
```

- [ ] **Step 2: 치환 검증**

```bash
grep -rn "github.com/manamana32321/homelab/health-hub" backend/
# Expected: no matches
grep -rn "github.com/manamana32321/health-hub/backend" backend/ | head
# Expected: go.mod + 7개 .go 파일
```

- [ ] **Step 3: Go 빌드 검증**

```bash
cd /tmp/health-hub-split/health-hub-working/backend
go build ./...
```

Expected: 에러 없이 완료.

- [ ] **Step 4: Go 테스트 실행**

```bash
cd /tmp/health-hub-split/health-hub-working/backend
go test ./...
```

Expected: PASS (기존 handlers_test.go 동작).

- [ ] **Step 5: 커밋**

```bash
cd /tmp/health-hub-split/health-hub-working
git add backend/
git commit -m "refactor(backend): update Go module path after repo split

Module path: github.com/manamana32321/homelab/health-hub
           → github.com/manamana32321/health-hub/backend"
```

---

## Task 4: CI workflow 새 경로로 수정

**Files:**
- Create: `/tmp/health-hub-split/health-hub-working/.github/workflows/backend-image.yaml`
- Create: `/tmp/health-hub-split/health-hub-working/.github/workflows/dashboard-image.yaml`

- [ ] **Step 1: backend workflow 작성**

```bash
mkdir -p /tmp/health-hub-split/health-hub-working/.github/workflows
```

Create `backend-image.yaml`:

```yaml
name: backend-image

on:
  push:
    branches: [main]
    paths:
      - backend/**
      - .github/workflows/backend-image.yaml
  workflow_dispatch:

env:
  IMAGE: ghcr.io/${{ github.repository_owner }}/health-hub

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.24"
          cache-dependency-path: backend/go.sum
      - name: Run tests
        working-directory: backend
        run: go test ./... -v -count=1

  build-push:
    runs-on: ubuntu-latest
    needs: test
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v4
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/build-push-action@v6
        with:
          context: backend
          push: true
          tags: |
            ${{ env.IMAGE }}:latest
            ${{ env.IMAGE }}:${{ github.sha }}
```

- [ ] **Step 2: dashboard workflow 작성**

Create `dashboard-image.yaml`:

```yaml
name: dashboard-image

on:
  push:
    branches: [main]
    paths:
      - dashboard/**
      - .github/workflows/dashboard-image.yaml
  workflow_dispatch:

env:
  IMAGE: ghcr.io/${{ github.repository_owner }}/health-hub-dashboard

jobs:
  build-push:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v4
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/build-push-action@v6
        with:
          context: dashboard
          push: true
          build-args: |
            NEXT_PUBLIC_API_TOKEN=${{ secrets.HEALTH_HUB_API_TOKEN }}
          tags: |
            ${{ env.IMAGE }}:latest
            ${{ env.IMAGE }}:${{ github.sha }}
```

- [ ] **Step 3: yaml 문법 검증**

```bash
cd /tmp/health-hub-split/health-hub-working
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/backend-image.yaml'))"
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/dashboard-image.yaml'))"
```

Expected: 에러 없음.

- [ ] **Step 4: 커밋**

```bash
cd /tmp/health-hub-split/health-hub-working
git add .github/workflows/
git commit -m "ci: add backend/dashboard image workflows for monorepo layout"
```

---

## Task 5: 루트 레벨 메타 파일 작성

**Files:**
- Create: `/tmp/health-hub-split/health-hub-working/README.md`
- Create: `/tmp/health-hub-split/health-hub-working/CLAUDE.md`
- Create: `/tmp/health-hub-split/health-hub-working/.gitignore`

- [ ] **Step 1: README.md 작성**

```markdown
# Health Hub

개인 건강 데이터 중앙 관리 시스템. Samsung Health + Galaxy Watch4 → Health Connect → Tasker → HTTP POST 파이프라인.

## 구조

- `backend/` — Go API 서버 + MCP 서버 (Claude Desktop 연결용)
- `dashboard/` — Next.js 16 시각화 대시보드

## 배포

K8s 매니페스트 및 ArgoCD Application은 [manamana32321/homelab](https://github.com/manamana32321/homelab) 레포의 [k8s/health-hub/](https://github.com/manamana32321/homelab/tree/main/k8s/health-hub)에서 관리.

이미지 빌드는 이 레포의 GitHub Actions에서 수행, ArgoCD Image Updater가 `digest` 전략으로 자동 배포.

- Production: https://health.json-server.win
- DB: TimescaleDB (PostgreSQL extension)
```

- [ ] **Step 2: CLAUDE.md 작성**

```markdown
# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Repository Overview

Health Hub — Samsung Health 생태계 데이터를 Health Connect + Tasker로 폰에서 서버로 push받아 TimescaleDB에 저장, Grafana/Next.js 대시보드로 시각화하는 개인 건강 관리 시스템.

## Structure

- `backend/` — Go 1.24, pgx/v5 + mark3labs/mcp-go. `cmd/server` (HTTP API, :8080) + `cmd/mcp` (MCP stdio, :8081 SSE 옵션)
- `dashboard/` — Next.js 16 (pnpm). Authentik forward auth로 보호

## Deployment

- **이미지**: GHCR public (`ghcr.io/manamana32321/health-hub`, `health-hub-dashboard`)
- **매니페스트**: [manamana32321/homelab](https://github.com/manamana32321/homelab) 레포의 `k8s/health-hub/manifests/`
- **ArgoCD**: `health-hub` Application, namespace `health-hub`, Image Updater `digest` 전략
- **DB**: TimescaleDB StatefulSet, `longhorn-ssd` 5Gi PVC
- **백업**: S3 `health-backup-json-server`, daily 02:00 KST (CronJob)

## Data Pipeline

Galaxy Watch4 → Samsung Health → Health Connect → Tasker → HTTP POST → backend `/api/samsung-health/webhook` → TimescaleDB

수집: 걸음수, 심박수, 수면, 운동, 체중, SpO2, 체성분(fat/lean/skeletal), 음식, 수분, 거리, 칼로리
미수집: 스트레스(Samsung 독점), 마음챙김(스킵)

## Conventions

- Go module: `github.com/manamana32321/health-hub/backend`
- 스키마 변경 시 `backend/internal/db/migrations/` 추가 + `db.go` embed에 등록 필수
- 도메인: health.json-server.win (Cloudflare proxied=true)
```

- [ ] **Step 3: .gitignore 작성**

```gitignore
# Go
backend/*.out
backend/health-hub
backend/health-hub-mcp

# Node
dashboard/node_modules/
dashboard/.next/
dashboard/out/
dashboard/.env*.local

# OS
.DS_Store
*.log

# IDE
.vscode/
.idea/
```

- [ ] **Step 4: 커밋**

```bash
cd /tmp/health-hub-split/health-hub-working
git add README.md CLAUDE.md .gitignore
git commit -m "docs: add README, CLAUDE.md, .gitignore for standalone repo"
```

---

## Task 6: 새 GitHub 레포 생성 및 push

**Files:**
- GitHub: `manamana32321/health-hub` (private)

- [ ] **Step 1: 사용자 승인 체크포인트**

**🛑 STOP — 사용자에게 확인:**

- 새 레포 `manamana32321/health-hub`를 **private**으로 생성합니다.
- 커밋 N개 + 파일 M개가 push됩니다.
- 이 단계는 되돌리기 어려움 (레포 삭제는 가능하나 GitHub Archive가 남음).

사용자 승인 후 Step 2 진행.

- [ ] **Step 2: 레포 생성**

```bash
gh repo create manamana32321/health-hub \
  --private \
  --description "개인 건강 데이터 중앙 관리 시스템 (Go + Next.js + TimescaleDB)"
```

Expected: `✓ Created repository manamana32321/health-hub on GitHub`

- [ ] **Step 3: remote 설정 + push**

```bash
cd /tmp/health-hub-split/health-hub-working
git remote add origin https://github.com/manamana32321/health-hub.git
git branch -M main
git push -u origin main
```

Expected: filter-repo 히스토리 + Task 3-5에서 추가한 커밋 전부 push 성공.

- [ ] **Step 4: GitHub UI 검증**

브라우저로 `https://github.com/manamana32321/health-hub` 접속:
- [ ] `backend/`, `dashboard/` 디렉토리 보임
- [ ] README.md 렌더링 정상
- [ ] Actions 탭에 workflow 2개 등록
- [ ] Settings → Collaborators → private 확인

- [ ] **Step 5: CI 첫 실행 트리거 (수동)**

Actions 탭 → backend-image → "Run workflow" → main → 실행. 완료 후:

```bash
gh run list --repo manamana32321/health-hub --limit 5
```

Expected: backend-image + dashboard-image 둘 다 `completed success`.

- [ ] **Step 6: GHCR 이미지 확인**

```bash
gh api /users/manamana32321/packages/container/health-hub/versions --jq '.[0].metadata.container.tags'
```

Expected: 최신 digest + `latest` tag 포함. 기존 이미지와 동일 이름이므로 ArgoCD가 새 digest를 감지하면 pull 정상.

---

## Task 7: homelab cleanup PR

**Files (homelab repo):**
- Delete: `health-hub/` (전체 디렉토리)
- Delete: `health-hub-dashboard/` (전체 디렉토리)
- Delete: `.github/workflows/health-hub-image.yaml`
- Delete: `.github/workflows/health-hub-dashboard-image.yaml`
- Modify: `CLAUDE.md` (만약 health-hub 관련 섹션이 있다면 링크 업데이트)

- [ ] **Step 1: 워크트리 생성**

```bash
cd /home/json/homelab-worktrees/main
git fetch origin main
git worktree add ~/homelab-worktrees/health-hub-repo-split -b chore/health-hub-repo-split origin/main
cd ~/homelab-worktrees/health-hub-repo-split
```

- [ ] **Step 2: 환경 파일 복사** (글로벌 룰 준수)

```bash
cp /home/json/homelab-worktrees/main/.envrc.local ./ 2>/dev/null || true
find /home/json/homelab-worktrees/main -maxdepth 3 \( -name '.env' -o -name '.env.local' -o -name '*.tfvars' \) -not -path '*/node_modules/*' | while read f; do
  rel="${f#/home/json/homelab-worktrees/main/}"
  mkdir -p "$(dirname "$rel")"
  cp "$f" "$rel"
done
direnv allow
```

- [ ] **Step 3: homelab.code-workspace에 폴더 엔트리 추가**

```bash
# 파일 위치 확인 후 수동 편집 또는 사용자 확인
ls ~/homelab.code-workspace 2>/dev/null || find ~ -name "homelab.code-workspace" -not -path "*/node_modules/*" 2>/dev/null | head -3
```

엔트리 추가 (JSON):
```json
{ "name": "homelab · health-hub-repo-split", "path": "homelab-worktrees/health-hub-repo-split" }
```

- [ ] **Step 4: 분리 대상 파일 삭제**

```bash
cd ~/homelab-worktrees/health-hub-repo-split
git rm -r health-hub/ health-hub-dashboard/
git rm .github/workflows/health-hub-image.yaml .github/workflows/health-hub-dashboard-image.yaml
```

- [ ] **Step 5: CLAUDE.md 확인**

```bash
grep -n "health-hub" CLAUDE.md || echo "no mention, skip"
```

만약 언급이 있다면 `[manamana32321/health-hub](https://github.com/manamana32321/health-hub)` 링크로 업데이트.

- [ ] **Step 6: k8s 매니페스트 영향 없음 확인**

```bash
ls k8s/health-hub/manifests/
cat k8s/argocd/applications/apps/health-hub.yaml | head -20
```

Expected:
- 매니페스트 파일 9개 그대로 존재
- Application CR의 `repoURL`은 `https://github.com/manamana32321/homelab` 유지 (매니페스트 위치는 homelab)
- image-list annotation도 그대로

- [ ] **Step 7: 커밋**

```bash
cd ~/homelab-worktrees/health-hub-repo-split
git add -A
git status
git commit -m "chore(health-hub): split app code to manamana32321/health-hub

앱 코드(Go backend + Next.js dashboard)를 별도 private 레포로 분리.
k8s 매니페스트 및 ArgoCD Application은 homelab에 유지.

- 삭제: health-hub/, health-hub-dashboard/, workflow 2개
- 유지: k8s/health-hub/, argocd app, grafana dashboard
- 히스토리: git filter-repo로 새 레포에 보존

Image Updater는 write-back-method: argocd 사용 중이라 영향 없음.
GHCR 이미지 이름 동일 (ghcr.io/manamana32321/health-hub[-dashboard]),
visibility public 유지 — imagePullSecret 불필요.

New repo: https://github.com/manamana32321/health-hub"
```

- [ ] **Step 8: push 및 PR 생성**

```bash
git push -u origin chore/health-hub-repo-split
gh pr create --title "chore(health-hub): split app code to separate repo" --body "$(cat <<'EOF'
## Summary

- 앱 코드(`health-hub/`, `health-hub-dashboard/`)를 [manamana32321/health-hub](https://github.com/manamana32321/health-hub) private monorepo로 분리
- 히스토리는 `git filter-repo`로 보존
- k8s 매니페스트/ArgoCD app/Grafana 대시보드는 homelab에 유지

## Impact

- **ArgoCD Image Updater**: 영향 없음 (`write-back-method: argocd` + GHCR 이미지 이름 동일)
- **K3s**: 영향 없음 (GHCR 패키지 visibility public 유지)
- **배포**: 새 이미지는 새 레포 CI에서 build → GHCR push → Image Updater가 digest 감지 → Application CR 업데이트

## Verification checklist

- [ ] 새 레포 CI 첫 빌드 성공 (https://github.com/manamana32321/health-hub/actions)
- [ ] GHCR 최신 digest 업데이트됨
- [ ] ArgoCD가 health-hub Application sync 성공
- [ ] health.json-server.win 응답 정상
- [ ] MCP sidecar 동작 확인 (Claude Desktop에서 `/mcp`)

EOF
)"
```

---

## Task 8: End-to-end 검증

**Files:** 실행만, 수정 없음

- [ ] **Step 1: 새 레포 CI 성공 확인**

```bash
gh run list --repo manamana32321/health-hub --limit 10
```

Expected: backend-image + dashboard-image 최신 실행 `completed success`.

- [ ] **Step 2: GHCR 새 digest 확인**

```bash
gh api /users/manamana32321/packages/container/health-hub/versions --jq '.[0] | {created_at, tags: .metadata.container.tags}'
gh api /users/manamana32321/packages/container/health-hub-dashboard/versions --jq '.[0] | {created_at, tags: .metadata.container.tags}'
```

Expected: `created_at`이 Task 6 Step 5 이후.

- [ ] **Step 3: ArgoCD Application sync 상태**

```bash
kubectl --context json get application health-hub -n argocd -o jsonpath='{.status.sync.status} {.status.health.status}{"\n"}'
kubectl --context json get pods -n health-hub
```

Expected: `Synced Healthy`, pod `Running`. Image Updater가 새 digest 반영했으면 Application manifest의 image annotation도 업데이트됨:

```bash
kubectl --context json get application health-hub -n argocd -o jsonpath='{.metadata.annotations.argocd-image-updater\.argoproj\.io/git-branch}{"\n"}'
```

- [ ] **Step 4: 서비스 헬스체크**

```bash
curl -sfI https://health.json-server.win/ | head -3
curl -sf https://health.json-server.win/api/health | jq .
```

Expected: 200 OK, API `{"status": "ok"}` 또는 유사 응답.

- [ ] **Step 5: MCP sidecar 확인** (사용자 수동)

Claude Desktop → MCP 서버 목록에서 health-hub 도구 호출 (`get_today_steps` 등).

- [ ] **Step 6: 메모리 업데이트**

`~/.claude/projects/-home-json-homelab-worktrees-main/memory/project_health_hub.md`:
- "`health-hub/` (homelab 레포 내)" → "[manamana32321/health-hub](https://github.com/manamana32321/health-hub) (private monorepo: backend/ + dashboard/)"
- Phase 2 표기 제거 (이미 완료)
- k8s 매니페스트는 homelab 유지 명시

`MEMORY.md` 변경 없음 (링크는 그대로).

- [ ] **Step 7: 임시 작업 공간 정리**

```bash
rm -rf /tmp/health-hub-split
```

- [ ] **Step 8: 워크트리 정리 (PR 머지 후)**

```bash
git worktree remove ~/homelab-worktrees/health-hub-repo-split
git branch -d chore/health-hub-repo-split
```

homelab.code-workspace에서 해당 folder 엔트리 제거.

---

## 실패 시나리오 및 복구

### 시나리오 A: filter-repo 결과 검증 실패 (Task 2)

**증상:** 파일 수 불일치, 관련 없는 커밋 포함 등.

**복구:** `/tmp/health-hub-split/` 삭제 후 Task 2부터 재실행. homelab 원본 영향 없음.

### 시나리오 B: 새 레포 CI 실패 (Task 6 Step 5)

**증상:** backend-image 또는 dashboard-image workflow 실패.

**복구:**
- Go 모듈 경로 치환 누락 → Task 3 재실행 후 force push (새 레포만이라 안전)
- Dashboard `NEXT_PUBLIC_API_TOKEN` secret 누락 → 새 레포 Settings → Secrets and variables → Actions에서 추가
- 이미지 빌드 실패 → 로컬에서 `cd backend && docker build .` 또는 `cd dashboard && docker build .`로 재현 후 수정

### 시나리오 C: ArgoCD Image Updater가 새 digest를 못 받음 (Task 8 Step 3)

**증상:** 새 이미지 push됐는데 Application CR의 이미지 annotation 미업데이트.

**원인:** Image Updater는 GHCR를 polling. 기본 interval 2분. 즉시 강제하려면:

```bash
kubectl --context json rollout restart deployment -n argocd argocd-image-updater
```

여전히 실패 시 Image Updater 로그 확인:

```bash
kubectl --context json logs -n argocd deployment/argocd-image-updater --tail=100
```

### 시나리오 D: homelab PR 머지 후 k8s sync 장애

**증상:** cleanup PR 머지 → ArgoCD가 무언가 삭제하려 함.

**예상 영향 없음** — 이 PR은 `health-hub/`, `health-hub-dashboard/`(소스), workflow 2개(CI)만 삭제. ArgoCD는 `k8s/health-hub/` 경로만 추적하므로 무영향.

검증:
```bash
kubectl --context json get application health-hub -n argocd -o jsonpath='{.status.sync.status}{"\n"}'
# Expected: Synced (변경 없음)
```

---

## Self-Review 체크리스트

- [x] **Spec coverage**: 앱 코드 분리 ✅ (Task 2-6), 히스토리 보존 ✅ (Task 2 filter-repo), monorepo 구조 ✅ (Task 2 path-rename), k8s 매니페스트 유지 ✅ (Task 7 — 삭제 대상에서 명시적 제외), GHCR public 유지 ✅ (Task 6에서 GHCR 설정 변경 없음)
- [x] **Placeholder scan**: TBD/TODO/"similar to" 없음. 모든 코드 블록 완전 작성
- [x] **Type consistency**: Go 모듈 경로 `github.com/manamana32321/health-hub/backend` 전 태스크 일관. 이미지 이름 `ghcr.io/manamana32321/health-hub[-dashboard]` 일관
- [x] **Reversibility**: Task 1-5는 임시 공간에서만 작업, Task 6부터 외부 영향. Step 6.1에 사용자 승인 체크포인트 명시

## 다음 단계

플랜 저장 완료: [docs/plans/2026-04-19-health-hub-repo-split.md](docs/plans/2026-04-19-health-hub-repo-split.md)

두 가지 실행 옵션:

1. **Subagent-Driven (추천)** — 태스크별 fresh subagent 디스패치, 태스크 간 review. 파일 이동/경로 치환 등 독립 작업이 많아 병렬화 이득은 제한적이지만 컨텍스트 관리에 유리.
2. **Inline Execution** — 현재 세션에서 executing-plans로 순차 실행. filter-repo 출력 등을 실시간 확인하며 진행.
