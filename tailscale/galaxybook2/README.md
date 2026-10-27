# galaxybook2 (Windows) — 집에서 subnet route 끄기

기본 게이트웨이 MAC 이 집 공유기(`b0:38:6c:39:41:2b`, `br-lan`)면 `tailscale set --accept-routes=false`, 아니면 `true`.

| 파일 | 위치 | 트리거 |
| --- | --- | --- |
| `tailscale-home-routes.ps1` | `%LOCALAPPDATA%\homelab\` | |
| `tailscale-home-routes.xml` | 작업 스케줄러 `\tailscale-home-routes` | 로그온, `Microsoft-Windows-NetworkProfile/Operational` 10000·10001 (+5초) |

## 설치

작업 등록만 관리자 권한. 실행은 `InteractiveToken` / `LeastPrivilege`.

```powershell
$dir = Join-Path $env:LOCALAPPDATA 'homelab'
New-Item -ItemType Directory -Force $dir | Out-Null
Copy-Item tailscale-home-routes.ps1, tailscale-home-routes.xml $dir -Force
```

관리자 PowerShell:

```powershell
Register-ScheduledTask -TaskName tailscale-home-routes -User "$env:USERDOMAIN\$env:USERNAME" -Force `
  -Xml (Get-Content -Raw -Encoding UTF8 "$env:LOCALAPPDATA\homelab\tailscale-home-routes.xml")
```

## 확인

```powershell
& 'C:\Program Files\Tailscale\tailscale.exe' debug prefs | Select-String RouteAll
(Find-NetRoute -RemoteIPAddress 192.168.0.44 | Select-Object -Last 1).InterfaceAlias
```

| 함정 | 증상 | 대응 |
| --- | --- | --- |
| XML `encoding="UTF-8"` 선언 | `Register-ScheduledTask` (1,40) 인코딩 전환 오류 | 선언 없음 |
| 일반 권한 등록 | `0x80070005` | 관리자 PowerShell 로 등록 |
| 공유기 교체 | 항상 `accept-routes=true` | `$HomeGatewayMac` 갱신 |
