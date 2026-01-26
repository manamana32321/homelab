# Factorio Admin Dashboard 설계

## 개요

Factorio 서버를 웹에서 관리할 수 있는 커스텀 대시보드 개발

## 레포지토리

- **위치**: `manamana32321/factorio-admin` (별도 레포지토리)
- **인증**: BetterAuth (자체 프로바이더)

## 사용자 요구사항

| 기능                                           | 기술적 가능성 | 구현 방법                           |
| ---------------------------------------------- | ------------- | ----------------------------------- |
| 플레이어 컨트롤 (리스트, 밴, 킥, 승격, 귓속말) | ✅ 완전 지원  | rconAPI REST 엔드포인트             |
| 맵 세이브 파일 CRUD                            | ✅ 가능       | PVC 접근 + RCON /save               |
| 맵 생성 옵션 선택                              | ⚠️ 제한적     | values.yaml 수정 → Pod 재시작       |
| 맵 미리보기/스크린샷                           | ❌ 불가       | 헤드리스 서버는 그래픽 미로드       |
| 세이브 백업 (자동/수동)                        | ✅ 가능       | CronJob + PVC 접근                  |
| 게임 로그 보기                                 | ⚠️ 폴링 필요  | RCON 연결 시 히스토리 + 주기적 폴링 |
| 실시간 대시보드 (전력/생산/공해)               | ⚠️ Lua 필요   | Lua 명령 실행 후 파싱               |
| 콘솔 입력/출력                                 | ✅ 완전 지원  | rconAPI /console/command            |
| 실시간 맵 보기                                 | ❌ 불가       | 헤드리스 서버 제약 (모드도 불가)    |

## 아키텍처

```
┌─────────────────────────────────────────────────────────────────┐
│                         Kubernetes (factorio namespace)          │
│                                                                  │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────────┐ │
│  │   Factorio   │     │  rconAPI     │     │  Admin Dashboard │ │
│  │   Server     │◄────│  (Go)        │◄────│  (NextJS)        │ │
│  │  :34197/UDP  │RCON │  :8080       │HTTP │  :3000           │ │
│  │  :27015/TCP  │     │  ClusterIP   │     │  ClusterIP       │ │
│  └──────┬───────┘     └──────────────┘     └────────┬─────────┘ │
│         │                                           │            │
│         └──────────────┬────────────────────────────┘            │
│                        │                                         │
│                   ┌────▼────┐                                    │
│                   │   PVC   │ (saves, mods, backups)             │
│                   │   4Gi   │                                    │
│                   └─────────┘                                    │
└─────────────────────────────────────────────────────────────────┘
                              │
                        Traefik Ingress
                              │
               ┌──────────────┴──────────────┐
               │                              │
    factorio-admin.json-server.win    factorio.json-server.win
         (HTTPS, OAuth2)                  (UDP, 게임 접속)
```

## 기술 스택

- **Frontend**: NextJS 14 (App Router) + TailwindCSS + shadcn/ui
- **Backend**: NextJS API Routes (rconAPI 프록시)
- **인증**: BetterAuth (GitHub OAuth 자체 구현)
- **DB**: SQLite (BetterAuth 세션/유저 저장, PVC 내)
- **배포**: Docker → GitHub Container Registry → ArgoCD
- **모니터링**: Prometheus metrics export (선택)

## 파일 구조

```text
factorio-admin/                    # manamana32321/factorio-admin 레포지토리
├── src/
│   ├── app/
│   │   ├── layout.tsx             # 공통 레이아웃
│   │   ├── page.tsx               # 대시보드 메인
│   │   ├── players/page.tsx       # 플레이어 관리
│   │   ├── saves/page.tsx         # 세이브 파일 관리
│   │   ├── console/page.tsx       # RCON 콘솔
│   │   ├── backups/page.tsx       # 백업 관리
│   │   ├── login/page.tsx         # 로그인 페이지
│   │   └── api/
│   │       ├── auth/[...all]/route.ts  # BetterAuth 핸들러
│   │       ├── rcon/route.ts      # rconAPI 프록시
│   │       ├── saves/route.ts     # 세이브 파일 API
│   │       └── backups/route.ts   # 백업 API
│   ├── components/
│   │   ├── PlayerList.tsx
│   │   ├── Console.tsx
│   │   ├── SaveManager.tsx
│   │   └── StatsCard.tsx
│   └── lib/
│       ├── auth.ts                # BetterAuth 설정
│       ├── auth-client.ts         # BetterAuth 클라이언트
│       ├── rcon.ts                # rconAPI 클라이언트
│       └── factorio.ts            # Lua 명령 헬퍼
├── Dockerfile
├── k8s/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── ingress.yaml
│   └── sealed-secret.yaml         # GitHub OAuth 시크릿
└── package.json
```

## Kubernetes 배포 구성

### 1. rconAPI 배포 (values.yaml 추가)

```yaml
# k8s/factorio/values.yaml에 추가
rconAPI:
  enabled: true
  type: ClusterIP
  httpPort: 8080
```

### 2. Admin Dashboard Deployment

```yaml
# k8s/factorio-admin/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: factorio-admin
  namespace: factorio
spec:
  replicas: 1
  selector:
    matchLabels:
      app: factorio-admin
  template:
    metadata:
      labels:
        app: factorio-admin
    spec:
      containers:
        - name: admin
          image: ghcr.io/manamana32321/factorio-admin:latest
          ports:
            - containerPort: 3000
          env:
            - name: RCON_API_URL
              value: "http://factorio-factorio-server-charts-rcon-api:8080"
            - name: RCON_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: factorio-rcon-password
                  key: rconpw
          volumeMounts:
            - name: factorio-data
              mountPath: /factorio
              readOnly: false
      volumes:
        - name: factorio-data
          persistentVolumeClaim:
            claimName: factorio-factorio-server-charts-datadir
```

### 3. Ingress (BetterAuth 자체 인증)

```yaml
# k8s/factorio-admin/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: factorio-admin
  namespace: factorio
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-dns01
spec:
  ingressClassName: traefik
  tls:
    - hosts:
        - factorio-admin.json-server.win
      secretName: factorio-admin-tls
  rules:
    - host: factorio-admin.json-server.win
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: factorio-admin
                port:
                  number: 3000
```

### 4. BetterAuth 설정

```typescript
// src/lib/auth.ts
import { betterAuth } from "better-auth";
import { Database } from "better-sqlite3";

export const auth = betterAuth({
  database: new Database("/factorio/auth.db"),
  socialProviders: {
    github: {
      clientId: process.env.GITHUB_CLIENT_ID!,
      clientSecret: process.env.GITHUB_CLIENT_SECRET!,
    },
  },
  // 허용된 사용자만 접근 (화이트리스트)
  user: {
    additionalFields: {
      isAdmin: { type: "boolean", defaultValue: false },
    },
  },
});
```

## 주요 기능 구현 상세

### 1. 플레이어 관리

```typescript
// src/lib/rcon.ts
const RCON_API = process.env.RCON_API_URL;

export async function getPlayers() {
  const res = await fetch(`${RCON_API}/api/v2/factorio/players`);
  return res.json();
}

export async function kickPlayer(name: string, reason?: string) {
  await fetch(`${RCON_API}/api/v2/factorio/players/${name}/kick`, {
    method: "POST",
    body: JSON.stringify({ reason }),
  });
}

export async function banPlayer(name: string, reason?: string) {
  await fetch(`${RCON_API}/api/v2/factorio/players/${name}/ban`, {
    method: "POST",
    body: JSON.stringify({ reason }),
  });
}

export async function promotePlayer(name: string) {
  await fetch(`${RCON_API}/api/v2/factorio/players/${name}/promote`, {
    method: "POST",
  });
}

export async function whisperPlayer(name: string, message: string) {
  await fetch(`${RCON_API}/api/v2/factorio/players/${name}/whisper`, {
    method: "POST",
    body: JSON.stringify({ message }),
  });
}
```

### 2. 콘솔

```typescript
// src/app/api/rcon/route.ts
export async function POST(req: Request) {
  const { command } = await req.json();

  const res = await fetch(
    `${RCON_API}/api/v2/factorio/console/command/${encodeURIComponent(command)}`,
    { method: "POST" },
  );

  return Response.json(await res.json());
}
```

### 3. 세이브 파일 관리

```typescript
// src/app/api/saves/route.ts
import { readdir, readFile, unlink, copyFile } from "fs/promises";
import path from "path";

const SAVES_DIR = "/factorio/saves";

export async function GET() {
  const files = await readdir(SAVES_DIR);
  const saves = await Promise.all(
    files
      .filter((f) => f.endsWith(".zip"))
      .map(async (f) => {
        const stat = await stat(path.join(SAVES_DIR, f));
        return { name: f, size: stat.size, modified: stat.mtime };
      }),
  );
  return Response.json(saves);
}

export async function DELETE(req: Request) {
  const { name } = await req.json();
  await unlink(path.join(SAVES_DIR, name));
  return Response.json({ success: true });
}
```

### 4. 백업 시스템

```typescript
// src/app/api/backups/route.ts
const BACKUPS_DIR = "/factorio/backups";

export async function POST(req: Request) {
  const { saveName } = await req.json();
  const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
  const backupName = `${saveName}_${timestamp}.zip`;

  await copyFile(
    path.join(SAVES_DIR, `${saveName}.zip`),
    path.join(BACKUPS_DIR, backupName),
  );

  return Response.json({ backup: backupName });
}
```

### 5. 게임 통계 (Lua 명령)

```typescript
// src/lib/factorio.ts
export async function getProductionStats() {
  // Lua 명령으로 통계 쿼리
  const command = `/silent-command
    local stats = game.forces.player.item_production_statistics
    local result = {}
    for name, count in pairs(stats.input_counts) do
      table.insert(result, {name=name, produced=count})
    end
    rcon.print(game.table_to_json(result))
  `;

  const res = await sendCommand(command);
  return JSON.parse(res);
}

export async function getPowerStats() {
  const command = `/silent-command
    local surface = game.surfaces[1]
    local networks = surface.find_entities_filtered{type="electric-pole"}
    -- ... 전력망 통계 수집
    rcon.print(game.table_to_json(result))
  `;

  return sendCommand(command);
}
```

## 구현 단계

### Phase 1: 기본 인프라 (1일)

1. rconAPI 활성화 (values.yaml 수정)
2. NextJS 프로젝트 초기화
3. Docker 이미지 빌드 및 푸시
4. ArgoCD Application 생성

### Phase 2: 핵심 기능 (2일)

1. 플레이어 관리 UI
2. RCON 콘솔 UI
3. 세이브 파일 목록/삭제

### Phase 3: 백업 시스템 (1일)

1. 수동 백업 기능
2. 자동 백업 CronJob
3. 백업 복원 기능

### Phase 4: 고급 기능 (2일)

1. 게임 통계 대시보드 (Lua 연동)
2. 게임 로그 뷰어 (폴링)
3. UI 개선

## 검증 방법

1. rconAPI 연결 테스트

   ```bash
   curl http://factorio-factorio-server-charts-rcon-api:8080/api/v2/factorio/version
   ```

2. 웹 UI 접속

   ```
   https://factorio-admin.json-server.win
   ```

3. 플레이어 명령 테스트 (콘솔에서 /players)

4. 세이브 파일 목록 확인

5. 백업 생성 및 복원 테스트

## 제약 사항

- **실시간 맵 보기**: 헤드리스 서버는 그래픽을 로드하지 않아 스크린샷/맵 뷰어 불가능
- **맵 생성 미리보기**: 불가능
- **게임 통계**: Lua silent-command 활성화 필요 (server-settings.json)
- **PVC 접근**: Admin Dashboard Pod가 Factorio 서버와 같은 노드에서 실행되어야 함 (ReadWriteOnce)

## GitHub OAuth 앱 설정

1. GitHub > Settings > Developer settings > OAuth Apps > New OAuth App
2. 설정:
   - Application name: `Factorio Admin`
   - Homepage URL: `https://factorio-admin.json-server.win`
   - Authorization callback URL: `https://factorio-admin.json-server.win/api/auth/callback/github`
3. Client ID와 Secret을 SealedSecret으로 저장
