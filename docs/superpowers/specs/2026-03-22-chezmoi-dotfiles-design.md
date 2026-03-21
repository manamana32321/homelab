# chezmoi로 dotfiles 관리 세팅

> GitHub Issue: #66
> 작성일: 2026-03-22

## 목표

개인 dotfiles (Claude Code 설정, Git, Zsh)를 chezmoi + private GitHub repo로 선언적 관리.
WSL 개발 머신 1대 대상.

## 도구

- **chezmoi**: 단일 Go 바이너리, 템플릿/시크릿/dry-run 지원, 한 줄 부트스트랩
- **Remote**: `manamana32321/dotfiles` (private GitHub repo)

## 1차 스코프 관리 대상

| 파일 | chezmoi 관리 방식 | 설명 |
|------|------------------|------|
| `~/.claude/CLAUDE.md` | 일반 파일 (덮어쓰기) | 글로벌 인스트럭션 |
| `~/.claude/settings.json` | `modify_` 스크립트 | hooks/plugins 선언적, permissions 로컬 보존 |
| `~/.claude/hooks/worktree-guard.sh` | 일반 파일 | 워크트리 컨벤션 guard hook |
| `~/.gitconfig` | 일반 파일 | Git 사용자/credential 설정 |
| `~/.zshrc` | 일반 파일 | API 키 2개 (`GREPTILE_API_KEY`, `GOG_PASSPHRASE`) 삭제 후 관리 |

## 제외 대상 (.chezmoiignore)

화이트리스트 방식: `~/.claude/` 전체를 제외한 뒤 관리 대상만 허용. Claude Code가 새 파일/디렉토리를 추가해도 자동으로 무시됨.

```
# .chezmoiignore
# ~/.claude/ 하위: 화이트리스트 방식 (전체 제외 → 관리 대상만 허용)
.claude/**
!.claude/CLAUDE.md
!.claude/settings.json
!.claude/hooks
!.claude/hooks/**
```

## Source 디렉토리 구조

```
~/.local/share/chezmoi/            (= manamana32321/dotfiles repo)
├── .chezmoiignore                 ← 제외 패턴
├── dot_claude/
│   ├── CLAUDE.md
│   ├── modify_settings.json       ← jq 머지 스크립트 (아래 상세)
│   └── hooks/
│       └── worktree-guard.sh
├── dot_gitconfig
└── dot_zshrc
```

## `modify_settings.json` 상세

`settings.json`은 세 영역의 관리 특성이 다름:

| 영역 | 특성 | 전략 |
|------|------|------|
| `hooks` | 모든 머신 동일해야 함 | **source 값으로 교체** |
| `enabledPlugins` | 모든 머신 동일해야 함 | **source 값으로 교체** |
| `permissions.allow` | 머신별로 사용하며 누적됨 | **로컬 값 보존** |
| `permissions.additionalDirectories` | 머신별로 다름 | **로컬 값 보존** |

### 동작 흐름

```
chezmoi apply
  → modify_settings.json 실행
  → stdin: 현재 로컬 ~/.claude/settings.json
  → 스크립트가 jq로 머지:
     1. source에 정의된 hooks, enabledPlugins를 적용
     2. 로컬의 permissions (allow, additionalDirectories)는 그대로 유지
  → stdout: 머지된 JSON
  → ~/.claude/settings.json에 씀
```

### 스크립트 구현 (의사코드)

```bash
#!/bin/bash
# modify_settings.json
set -euo pipefail

# 로컬 현재 상태 (새 머신에서는 빈 입력 → 빈 객체로 초기화)
LOCAL=$(cat)
if [ -z "$LOCAL" ]; then
  LOCAL='{}'
fi

# source에서 관리하는 선언적 값들
HOOKS='{ "PreToolUse": [{ "matcher": "Bash", "hooks": [{ "type": "command", "command": "bash /home/json/.claude/hooks/worktree-guard.sh" }] }] }'
PLUGINS='{ ... }'  # 구현 시 현재 settings.json의 enabledPlugins 전체를 여기에 복사

# 머지: hooks/plugins는 교체, permissions는 보존
echo "$LOCAL" | jq \
  --argjson hooks "$HOOKS" \
  --argjson plugins "$PLUGINS" \
  '.hooks = $hooks | .enabledPlugins = $plugins'
```

## .zshrc 시크릿 처리

**삭제 대상:**
- `GREPTILE_API_KEY=i8gyu...` → 삭제 (필요 시 별도 경로에서 관리)
- `GOG_PASSPHRASE="1234"` → 삭제
- `GOG_ACCOUNT="manamana32321@gmail.com"` → 삭제 (이메일이지만 계정 정보)

삭제 후 `.zshrc`를 그대로 일반 파일로 chezmoi 관리.

## 부트스트랩 (새 머신)

```bash
# 1. chezmoi 설치 + repo 클론 + 적용
sh -c "$(curl -fsLS get.chezmoi.io)" -- init --apply manamana32321/dotfiles

# 2. 완료 — settings.json은 modify_ 스크립트가 빈 permissions로 초기화
```

`jq` 의존성: modify_ 스크립트 실행에 필요. 부트스트랩 전 `sudo apt install jq` 필요.

## 일상 워크플로우

```bash
# 일반 파일 (.zshrc, .gitconfig, CLAUDE.md, hooks/) 변경 후 source에 반영
chezmoi re-add

# 또는 chezmoi edit으로 직접 source 수정 → 자동 apply
chezmoi edit ~/.zshrc

# settings.json의 hooks/plugins 변경 시: modify_ 스크립트를 직접 수정
chezmoi edit ~/.claude/settings.json  # → modify_settings.json 스크립트를 에디터로 열림
# (permissions 변경은 로컬에서 자연 누적 — source 반영 불필요)

# 커밋 & 푸시
chezmoi git add .
chezmoi git commit -- -m "update zshrc aliases"
chezmoi git push
```

**주의**: `chezmoi re-add`는 일반 파일에만 적용됨. `modify_` 스크립트로 관리되는 `settings.json`은 `re-add` 불가 — 스크립트 자체를 수정해야 함.

## 설정 파일

1차 스코프에서는 `chezmoi.toml` 불필요 (age 미사용, 단일 머신). 추후 age 도입 시 `~/.config/chezmoi/chezmoi.toml`에 encryption 설정 추가.

## 추후 스코프 (이번 미포함)

- `~/.aws/config` (credentials 제외)
- `~/.kube/config` (age 암호화 필요 → 이때 age 설정 도입)
- Oh My Zsh 커스텀 설정 (`~/.oh-my-zsh/custom/`)
- `.envrc` 정리 (없으면 삭제 — 현재 홈 디렉토리에 `.envrc` 없음 확인 완료)
