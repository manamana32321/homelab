# chezmoi Dotfiles 관리 세팅 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** chezmoi + private GitHub repo로 dotfiles (Claude Code 설정, .gitconfig, .zshrc)를 선언적으로 관리

**Architecture:** chezmoi가 `~/.local/share/chezmoi/` (= GitHub repo)를 source of truth으로 사용. 일반 파일은 덮어쓰기, `settings.json`만 `modify_` 스크립트로 선택적 머지. `.chezmoiignore` 화이트리스트로 `~/.claude/` 하위 불필요 파일 자동 제외.

**Tech Stack:** chezmoi, jq, GitHub (private repo), bash

**Spec:** `docs/superpowers/specs/2026-03-22-chezmoi-dotfiles-design.md`

---

### Task 1: chezmoi 설치 및 초기화

**Files:**
- Create: `~/.local/share/chezmoi/` (chezmoi source directory)

- [ ] **Step 1: chezmoi 설치**

```bash
sh -c "$(curl -fsLS get.chezmoi.io)"
```

Expected: `~/.local/bin/chezmoi` 생성

- [ ] **Step 2: 설치 확인**

```bash
chezmoi --version
```

Expected: 버전 출력 (예: `chezmoi version v2.x.x`)

- [ ] **Step 3: jq 설치 확인**

```bash
jq --version
```

Expected: `jq-1.x`. 없으면 `sudo apt install jq`

- [ ] **Step 4: chezmoi init**

```bash
chezmoi init
```

Expected: `~/.local/share/chezmoi/` 디렉토리 생성

- [ ] **Step 5: 확인**

```bash
ls -la "$(chezmoi source-path)"
```

Expected: `~/.local/share/chezmoi` 경로, `.git/` 디렉토리 존재

---

### Task 2: .zshrc 시크릿 제거 및 추가

**Files:**
- Modify: `~/.zshrc` (API 키 3줄 삭제)

- [ ] **Step 1: .zshrc에서 시크릿 라인 삭제**

`~/.zshrc`에서 시크릿 3줄 삭제:

```bash
sed -i '/^export GOG_PASSPHRASE=/d; /^export GOG_ACCOUNT=/d; /^export GREPTILE_API_KEY=/d' ~/.zshrc
```

- [ ] **Step 2: 삭제 확인**

```bash
grep -n 'GOG_PASSPHRASE\|GOG_ACCOUNT\|GREPTILE_API_KEY' ~/.zshrc
```

Expected: 출력 없음

- [ ] **Step 3: chezmoi에 .zshrc 추가**

```bash
chezmoi add ~/.zshrc
```

Expected: `~/.local/share/chezmoi/dot_zshrc` 생성

- [ ] **Step 4: 추가 확인**

```bash
ls -la "$(chezmoi source-path)/dot_zshrc"
grep -c 'GREPTILE_API_KEY' "$(chezmoi source-path)/dot_zshrc"
```

Expected: 파일 존재, grep 카운트 0

- [ ] **Step 5: 커밋**

```bash
chezmoi git add .
chezmoi git commit -- -m "feat: add .zshrc (secrets removed)"
```

---

### Task 3: .gitconfig 추가

**Files:**
- Create: `~/.local/share/chezmoi/dot_gitconfig`

- [ ] **Step 1: chezmoi에 .gitconfig 추가**

```bash
chezmoi add ~/.gitconfig
```

Expected: `~/.local/share/chezmoi/dot_gitconfig` 생성

- [ ] **Step 2: 내용 확인**

```bash
chezmoi diff
```

Expected: 차이 없음 (source와 target 동일)

- [ ] **Step 3: 커밋**

```bash
chezmoi git add .
chezmoi git commit -- -m "feat: add .gitconfig"
```

---

### Task 4: Claude Code 일반 파일 추가 (CLAUDE.md, hooks/)

**Files:**
- Create: `~/.local/share/chezmoi/dot_claude/CLAUDE.md`
- Create: `~/.local/share/chezmoi/dot_claude/hooks/worktree-guard.sh`

- [ ] **Step 1: CLAUDE.md 추가**

```bash
chezmoi add ~/.claude/CLAUDE.md
```

Expected: `dot_claude/CLAUDE.md` 생성

- [ ] **Step 2: hooks/worktree-guard.sh 추가**

```bash
chezmoi add ~/.claude/hooks/worktree-guard.sh
```

Expected: `dot_claude/hooks/worktree-guard.sh` 생성

- [ ] **Step 3: source 구조 확인**

```bash
find "$(chezmoi source-path)/dot_claude" -type f
```

Expected:
```
.../dot_claude/CLAUDE.md
.../dot_claude/hooks/worktree-guard.sh
```

- [ ] **Step 4: 커밋**

```bash
chezmoi git add .
chezmoi git commit -- -m "feat: add Claude Code config (CLAUDE.md, hooks)"
```

---

### Task 5: .chezmoiignore 작성

**Files:**
- Create: `~/.local/share/chezmoi/.chezmoiignore`

- [ ] **Step 1: .chezmoiignore 파일 생성**

`~/.local/share/chezmoi/.chezmoiignore` 파일을 생성:

```
# ~/.claude/ 하위: 화이트리스트 방식 (전체 제외 → 관리 대상만 허용)
.claude/**
!.claude/CLAUDE.md
!.claude/settings.json
!.claude/hooks
!.claude/hooks/**
```

- [ ] **Step 2: 동작 확인 — managed 파일만 표시되는지**

```bash
chezmoi managed | grep '\.claude'
```

Expected (`settings.json`은 아직 추가 전이므로 Task 6 이후에 나타남):
```
.claude/CLAUDE.md
.claude/hooks/worktree-guard.sh
```

- [ ] **Step 3: 역방향 확인 — unmanaged에 claude 하위 파일 없는지**

```bash
chezmoi unmanaged --path-style absolute | grep '\.claude' | head -5
```

Expected: `.credentials.json`, `projects/`, `cache/` 등이 목록에 나와도 chezmoi가 관리하지 않으므로 정상. `chezmoi managed`에 없는 것이 핵심.

- [ ] **Step 4: 커밋**

```bash
chezmoi git add .
chezmoi git commit -- -m "feat: add .chezmoiignore (whitelist for ~/.claude/)"
```

---

### Task 6: modify_settings.json 스크립트 작성

**Files:**
- Create: `~/.local/share/chezmoi/dot_claude/modify_settings.json`

- [ ] **Step 1: 현재 settings.json에서 hooks와 enabledPlugins 값 추출**

```bash
jq '.hooks' ~/.claude/settings.json > /tmp/chezmoi-hooks.json
jq '.enabledPlugins' ~/.claude/settings.json > /tmp/chezmoi-plugins.json
```

Expected: 두 JSON 파일 생성

- [ ] **Step 2: modify_settings.json 스크립트 작성**

`~/.local/share/chezmoi/dot_claude/modify_settings.json` 파일을 생성:

```bash
#!/bin/bash
set -euo pipefail

# 로컬 현재 상태 (새 머신에서는 빈 입력 → 빈 객체로 초기화)
LOCAL=$(cat)
if [ -z "$LOCAL" ]; then
  LOCAL='{}'
fi

# source에서 관리하는 선언적 값들 (heredoc으로 가독성 확보)
HOOKS=$(cat <<'HOOKS_EOF'
<여기에 /tmp/chezmoi-hooks.json 내용 붙여넣기>
HOOKS_EOF
)

PLUGINS=$(cat <<'PLUGINS_EOF'
<여기에 /tmp/chezmoi-plugins.json 내용 붙여넣기>
PLUGINS_EOF
)

# 머지: hooks/plugins는 교체, permissions는 보존
echo "$LOCAL" | jq \
  --argjson hooks "$HOOKS" \
  --argjson plugins "$PLUGINS" \
  '.hooks = $hooks | .enabledPlugins = $plugins'
```

**주의:**
- `<여기에 ... 붙여넣기>` 부분을 Step 1에서 추출한 실제 JSON으로 교체할 것.
- `chezmoi add ~/.claude/settings.json`은 실행하지 말 것 — `modify_` 스크립트가 일반 소스 파일을 대체함.

- [ ] **Step 3: 실행 권한 부여**

```bash
chmod +x "$(chezmoi source-path)/dot_claude/modify_settings.json"
```

- [ ] **Step 4: dry-run으로 동작 확인**

```bash
chezmoi diff
```

Expected: `~/.claude/settings.json`에 대해 차이 없음 (hooks/plugins가 동일하므로). 차이가 있으면 스크립트의 JSON 값 확인.

- [ ] **Step 5: apply 테스트**

```bash
chezmoi apply --dry-run --verbose
```

Expected: `settings.json`이 modify 스크립트를 통해 처리됨, 실제 변경 없음

- [ ] **Step 6: 실제 apply**

```bash
chezmoi apply
```

- [ ] **Step 7: apply 후 settings.json 무결성 확인**

```bash
jq . ~/.claude/settings.json > /dev/null && echo "Valid JSON"
jq '.hooks | keys' ~/.claude/settings.json
jq '.enabledPlugins | keys | length' ~/.claude/settings.json
jq '.permissions.allow | length' ~/.claude/settings.json
```

Expected: "Valid JSON", hooks 키 존재, plugins 수 동일, permissions.allow 개수 동일 (304)

- [ ] **Step 8: 임시 파일 정리 및 커밋**

```bash
rm /tmp/chezmoi-hooks.json /tmp/chezmoi-plugins.json
chezmoi git add .
chezmoi git commit -- -m "feat: add modify_settings.json (merge hooks/plugins, preserve permissions)"
```

---

### Task 7: GitHub remote 연결

**Files:**
- Modify: `~/.local/share/chezmoi/.git/config` (remote 추가)

- [ ] **Step 1: private repo 생성**

```bash
gh repo create manamana32321/dotfiles --private
```

Expected: `https://github.com/manamana32321/dotfiles` 생성

- [ ] **Step 2: remote 추가 및 push**

```bash
chezmoi git remote add origin -- https://github.com/manamana32321/dotfiles.git
chezmoi git push -- -u origin main
```

Expected: 모든 커밋이 push됨

- [ ] **Step 3: GitHub에서 확인**

```bash
gh repo view manamana32321/dotfiles --json name,isPrivate,defaultBranchRef
```

Expected: `isPrivate: true`, `defaultBranchRef.name: main`

---

### Task 8: 전체 검증

- [ ] **Step 1: chezmoi verify (source와 target 일치 확인)**

```bash
chezmoi verify
```

Expected: 종료 코드 0 (차이 없음)

- [ ] **Step 2: managed 파일 목록 최종 확인**

```bash
chezmoi managed
```

Expected:
```
.claude/CLAUDE.md
.claude/hooks/worktree-guard.sh
.claude/settings.json
.gitconfig
.zshrc
```

- [ ] **Step 3: diff 확인**

```bash
chezmoi diff
```

Expected: 출력 없음 (모든 파일 동기화 완료)

- [ ] **Step 4: homelab repo에서 이슈 TODO 체크리스트 업데이트**

```bash
# 이슈 #66의 완료된 TODO 항목 확인 후 코멘트
gh issue comment 66 --repo manamana32321/homelab --body "1차 스코프 완료: chezmoi init, dotfiles 추가, modify_settings.json, GitHub remote 연결. repo: manamana32321/dotfiles (private)"
```
