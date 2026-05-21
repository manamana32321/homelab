# 마크 서버 클라이언트 설정 가이드

`mc.json-server.win` (Paper **1.21.11**, Java Edition) 접속 친구용 클라이언트 편의 모드 안내.

서버는 손댈 게 없습니다 — 미니맵·월드맵은 순수 클라이언트 모드이고, online-mode Paper 서버와 그대로 호환됩니다(치트 아님). 친구별로 한 번만 설치하면 됩니다.

## 미니맵 + 월드맵 (Xaero's)

**준비물**: Minecraft Java Edition **1.21.11** (서버 접속용으로 이미 필요한 버전)

### 1. Fabric 설치 (모드 로더)

1. <https://fabricmc.net/use/installer/> 에서 인스톨러를 받아 실행
2. **Client** 탭 → Minecraft Version을 **1.21.11** 선택 → **Install**
   - Minecraft 런처는 닫아둔 상태로 설치

### 2. mods 폴더 열기

- **Windows**: `Win + R` → `%appdata%\.minecraft` 입력 → 그 안의 `mods` 폴더 (없으면 새로 만들기)
- **macOS**: `~/Library/Application Support/minecraft/mods`

### 3. 모드 3개 다운로드 → `mods` 폴더에 넣기

아래 Modrinth 페이지에서 **게임 버전 `1.21.11` / 로더 `Fabric`** 으로 필터해 받으세요:

| 모드 | 역할 | 링크 |
|------|------|------|
| Fabric API | 필수 라이브러리 | <https://modrinth.com/mod/fabric-api> |
| Xaero's Minimap | 코너 미니맵 | <https://modrinth.com/mod/xaeros-minimap> |
| Xaero's World Map | 전체화면 지도 | <https://modrinth.com/mod/xaeros-world-map> |

받은 `.jar` 3개를 전부 `mods` 폴더에 넣습니다.

### 4. 실행

Minecraft 런처 → 왼쪽 아래 프로필을 **`fabric-loader-1.21.11`** 로 변경 → 플레이

## 사용법

- 우측 상단에 **미니맵 자동 표시**
- **M 키** — 전체화면 월드맵 (탐험한 곳이 기록됨)
- 다른 플레이어 위치 표시(레이더)가 부담되면 미니맵 설정에서 끌 수 있음

## 문제 해결

게임이 안 켜지면 대부분 버전 불일치입니다. Fabric 로더 · Fabric API · Xaero's 2개가 **전부 1.21.11**용인지 확인하세요.

---

> 이 문서가 가이드의 SSOT입니다. 배포용 PDF는 이 파일에서 생성합니다. 편의 모드를 추가하면 이 문서에 누적합니다.
