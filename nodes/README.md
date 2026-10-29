# 노드 OS 설정

클러스터 노드의 OS 레벨 설정. ArgoCD 로 배포되지 않고 **사람이 복사해서 적용**한다.

## netplan

`netplan/<노드>/` 는 그 노드의 `/etc/netplan/` 에 들어가는 파일이다.

| 노드 | 파일 | 내용 |
| --- | --- | --- |
| json-server-1 | `99-static.yaml` | `enp3s0` 고정 IP `192.168.0.27/24`, GW·DNS 모두 `192.168.0.1` |
| json-server-2 | `99-static.yaml` | `eno1` 고정 IP `192.168.0.46/24`, 동일 |
| raspi-1 | `99-wifi.yaml` | `wlan0` 무선 접속 (비밀번호는 `REPLACE_ME` 플레이스홀더) |

raspi-1 의 `eth0` 은 고정하지 않는다. 은퇴 노드라 그 IP 를 참조하는 곳이 없고, 공유기 DHCP 예약(`.37`)으로 주소가 이미 안정적이다. `wlan0` 은 메트릭 600 으로 유선 실패 시 예비 경로가 된다.

`50-cloud-init.yaml` 은 cloud-init 이 부팅마다 다시 쓰므로 **수정하지 않는다.** netplan 은 파일 이름 순서로 병합해서 뒤 파일이 이기므로 `99-` 로 덮는다.

렌더러는 노드마다 다르지만 netplan 이 상위 선언이라 같은 방식으로 다룬다 — json-server-1 과 raspi-1 은 `systemd-networkd`, json-server-2 는 `NetworkManager` 가 인터페이스를 쥐고 있다.

### 적용

```bash
scp nodes/netplan/<노드>/99-*.yaml json@<IP>:/tmp/
ssh json@<IP> 'sudo mv /tmp/99-*.yaml /etc/netplan/ && sudo chmod 600 /etc/netplan/99-*.yaml && sudo netplan generate'
```

`netplan apply` 는 네트워크를 재설정하면서 SSH 세션을 끊는다. 명령이 중간에 죽지 않게 떼어내서 실행한다.

```bash
ssh json@<IP> 'sudo nohup sh -c "sleep 1; netplan apply" >/dev/null 2>&1 &'
```

무선 비밀번호가 있는 파일은 적용 전에 `REPLACE_ME` 를 실제 값으로 바꾼다.

## 노드 IP

노드는 고정 IP 와 공유기 DHCP 예약을 **병행**한다. 고정 IP 가 실제 동작이고 예약은 이중 안전장치 겸 "이 주소는 쓰는 중" 이라는 기록이다. 공유기 쪽 예약은 `openwrt/config/dhcp` 에 있다.

DNS 는 **공유기(`192.168.0.1`)를 가리킨다.** ISP 가 주는 DNS 서버 주소는 공유기가 WAN DHCP 로 받아 dnsmasq 가 전달하므로, ISP 고유 값이 레포에 복제되지 않는다. 회선이 바뀌어도 노드 설정은 그대로다.

파드의 외부 이름 해석도 이 경로를 타고 간다 — CoreDNS 가 `forward . /etc/resolv.conf` 로 노드 리졸버에 넘긴다.

## 노드 공통 설정

| 항목 | 값 | json-server-1 | json-server-2 |
| --- | --- | --- | --- |
| 절전 target | `systemctl mask sleep.target suspend.target hibernate.target hybrid-sleep.target suspend-then-hibernate.target` | 2026-10-22 | 2026-10-22 |
| swap | `swapoff -a`, `/etc/fstab` swap 줄 주석, `/swap.img` 삭제 | 2026-10-22 | 2026-10-22 |
| 데스크톱 서비스 | `systemctl mask` `cups.socket` `cups.path` `cups` `cups-browsed` `ModemManager` `apport` `kerneloops` (존재하는 것만) | 2026-10-22 | 2026-10-22 |
| rsyslog | `apt-get purge rsyslog` | 2026-10-22 | 2026-10-22 |
| 전원 버튼 | `logind/power-key.conf` → `/etc/systemd/logind.conf.d/`, `systemctl reload systemd-logind` | 2026-10-22 | 2026-10-22 |
| NIC EEE | `udev/70-eee-off.rules` → `/etc/udev/rules.d/`, `udevadm control --reload` | 2026-10-22 | 2026-10-22 |
| journald 상한 | `journald/size.conf` → `/etc/systemd/journald.conf.d/`, `systemctl restart systemd-journald` | 2026-10-22 | 2026-10-22 |

## 키오스크 (json-server-2)

HDMI 모니터에 GNOME 자동 로그인(`kiosk`) → Chrome 전체화면 Grafana. RDP 로 같은 화면 공유.

`kiosk-screen auto` 가 1분마다 판단한다. 08~23시 **그리고** 폰이 집 Wi-Fi 에 붙어 있으면 켜짐, 아니면 꺼짐.

| 항목 | 값 |
| --- | --- |
| 재실 신호 | 공유기 `192.168.0.1:9100/metrics` 에 `mac="BE:4F:43:38:44:31"` (Z Flip5, SSID 별 고정 랜덤 MAC) |
| 외출 판정 | 마지막으로 보인 뒤 3분 (`$XDG_RUNTIME_DIR/kiosk-phone-seen` mtime) |
| 공유기 응답 실패 | 재실로 간주 (켜짐) |
| 폰 교체·MAC 변경 | j2·Coral 양쪽 `kiosk-screen` 의 `PHONE` 수정. 현재 값은 Loki `{service_name="openwrt", unit="dnsmasq-dhcp"} \|= "DHCPACK"` 의 호스트명으로 확인 |

| 레포 `nodes/kiosk/json-server-2/` | 노드 경로 | 반영 |
| --- | --- | --- |
| `sysusers-kiosk.conf` | `/etc/sysusers.d/kiosk.conf` | `systemd-sysusers` |
| `tmpfiles-kiosk.conf` | `/etc/tmpfiles.d/kiosk.conf` | `systemd-tmpfiles --create --remove` |
| `gdm-custom.conf` | `/etc/gdm3/custom.conf` | |
| `dconf-profile-user` | `/etc/dconf/profile/user` | `dconf update` |
| `dconf-00-kiosk` | `/etc/dconf/db/local.d/00-kiosk` | `dconf update` |
| `config.yaml` | `/etc/grafana-kiosk/config.yaml` | |
| `grafana-kiosk.service` `kiosk-screen.service` `kiosk-screen.timer` | `/etc/systemd/user/` | |
| `kiosk-screen` | `/usr/local/bin/kiosk-screen` (755) | |
| `j2-kiosk.rdp` | 클라이언트 (Windows) | |

전부 반영 후 kiosk `systemctl --user daemon-reload` → `systemctl restart gdm`. 최초 1회 `systemctl set-default graphical.target`.

| 레포 밖 | 위치 | 생성 |
| --- | --- | --- |
| `google-chrome-stable` | apt `https://dl.google.com/linux/chrome/deb/ stable main` | 키 `linux_signing_key.pub` → `/usr/share/keyrings/google-chrome.gpg` |
| `grafana-kiosk` v1.0.13 | `/usr/local/bin/grafana-kiosk` | GitHub release `grafana-kiosk.linux.amd64` |
| Grafana 비밀번호 | `/etc/grafana-kiosk/env` (`KIOSK_LOGIN_PASSWORD=`) | 로컬 사용자 `kiosk-json-server-2` (Viewer). Server Admin → Users → New user |
| RDP TLS | `/home/kiosk/.local/share/gnome-remote-desktop/rdp-tls.{crt,key}` | 아래 |
| RDP 자격 증명 | kiosk login 키링 (평문) | 아래 |
| RDP 비밀번호 (클라이언트) | WSL `~/.local/share/homelab/j2-rdp.env`, Windows `cmdkey TERMSRV/192.168.0.46` | |

```bash
U="sudo -u kiosk XDG_RUNTIME_DIR=/run/user/1500 DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1500/bus"
G=/home/kiosk/.local/share/gnome-remote-desktop
sudo -u kiosk openssl req -x509 -newkey rsa:2048 -nodes -days 3650 -subj /CN=json-server-2 -keyout $G/rdp-tls.key -out $G/rdp-tls.crt
$U grdctl rdp set-tls-cert $G/rdp-tls.crt
$U grdctl rdp set-tls-key $G/rdp-tls.key
$U grdctl rdp set-credentials kiosk '<RDP_PASS>'
$U grdctl rdp disable-view-only
$U grdctl rdp enable
```

수동 전환: `$U kiosk-screen on|off|auto` — 다음 1분 타이머가 `auto` 로 덮어쓴다. 오래 유지하려면 `$U systemctl --user stop kiosk-screen.timer`

## 키오스크 (mocha-snail)

Coral Dev Board, Mendel 5.2 (Debian 10), Wi-Fi `192.168.0.44`. Weston 5.0 위 Xwayland 로 Chrome 전체화면. GPU 미사용.

`kiosk-screen auto` 가 1분마다 판단한다. 재실 판정은 j2 와 같다 (`/run/kiosk-phone-seen`).

| 시간 | 재실 | 상태 |
| --- | --- | --- |
| 08~23시 | 있음 | Weston 실행 (키오스크) |
| 08~23시 | 없음 | Weston 정지 (빈 콘솔, 백라이트 켜짐) |
| 23~08시 | - | 절전 (suspend-to-RAM), 08:00:05 RTC 알람으로 복귀 |

| 레포 `nodes/kiosk/mocha-snail/` | 노드 경로 | 반영 |
| --- | --- | --- |
| `apt-debian-buster-archive.list` | `/etc/apt/sources.list.d/debian-buster-archive.list` | `apt-get update` |
| `apt-preferences-debian-buster-archive` | `/etc/apt/preferences.d/debian-buster-archive` | |
| `apt-99archive-no-valid-until` | `/etc/apt/apt.conf.d/99archive-no-valid-until` | |
| `coral-kiosk` | `/usr/local/bin/coral-kiosk` (755) | |
| `coral-kiosk.service` | `/etc/systemd/system/coral-kiosk.service` | `systemctl daemon-reload && systemctl enable --now coral-kiosk` |
| `env.example` | `/etc/coral-kiosk/env` | 키 채워서 생성 (아래 lifehub 키), `systemctl restart coral-kiosk` |
| `kiosk-screen` | `/usr/local/bin/kiosk-screen` (755) | |
| `kiosk-screen.service` `kiosk-screen.timer` | `/etc/systemd/system/` | `systemctl daemon-reload && systemctl enable kiosk-screen.service kiosk-screen.timer` |
| `tmpfiles-fan.conf` | `/etc/tmpfiles.d/coral-fan.conf` | `systemd-tmpfiles --create /etc/tmpfiles.d/coral-fan.conf` |
| `systemd-no-watchdog.conf` | `/etc/systemd/system.conf.d/no-watchdog.conf` | 재부팅 (한 번 켜진 imx2 watchdog 은 리셋 전까지 못 끔) |
| `weston.ini` | `/etc/xdg/weston/weston.ini` | `systemctl restart weston` |
| `weston-restart.conf` | `/etc/systemd/system/weston.service.d/restart.conf` | `systemctl daemon-reload` |
| `udev-kiosk-hotplug.rules` | `/etc/udev/rules.d/90-coral-kiosk-hotplug.rules` | `udevadm control --reload` |
| `coral-kiosk-hotplug.service` | `/etc/systemd/system/coral-kiosk-hotplug.service` | `systemctl daemon-reload` |
| `coral-kiosk-refit` | `/usr/local/bin/coral-kiosk-refit` (755) | |
| `nm-dispatcher-timesyncd` | `/etc/NetworkManager/dispatcher.d/90-timesyncd` (755, root) | |

| 레포 밖 | 위치 | 생성 |
| --- | --- | --- |
| 비활성 저장소 | `/etc/apt/sources.list.d/multistrap-{main,bsp,edgetpu}.list.disabled` | `mendel-linux.org` NXDOMAIN, `packages.cloud.google.com` 403 |
| 패키지 | `google-chrome-stable` `fonts-noto-cjk` `xdotool` `x11-utils` | `apt-get install` |
| Chrome 저장소 | `/etc/apt/sources.list.d/google-chrome.sources` | `google-chrome-stable` 설치 시 자동 |
| Wi-Fi | NM 프로필 `5G` | `nmcli con modify 5G wifi-sec.psk '<무선비밀번호>' 802-11-wireless.powersave 2` |
| 유선 | NM 프로필 `Wired connection 1` | `nmcli con down "Wired connection 1"` |
| 시간대 | `Asia/Seoul` | `timedatectl set-timezone Asia/Seoul` |

| 함정 | 증상 | 대응 |
| --- | --- | --- |
| `--kiosk` / 시작부터 fullscreen | X 상태는 정상, 모니터엔 바탕화면만 | `--app` 창 표시 후 `xdotool` F11 |
| Wayland 직접 실행 | `No Wayland shell found` (Weston 5.0 에 `xdg_wm_base` 없음) | `--ozone-platform=x11` |
| Wi-Fi 절전 | LAN RTT 55~160ms, 유휴 후 SSH 타임아웃 | `powersave 2` |
| Xwayland 지연 기동 | `Failed creating new xdo instance` | `xprop -root` 응답 대기 |
| 화면 끄기 | `fb0/blank`·`setterm --blank` 는 콘솔이 다시 켬. `card0-HDMI-A-1/status`=`off` 는 `HDMI/DP Cable Plug Out` 분당 ~280회 → 모니터 켜짐/꺼짐 반복 | 야간은 절전 (`rtcwake -m no` + `systemctl suspend`), 낮 외출은 Weston 정지 |
| 신호 끊기로 모니터 끄기 | Weston 정지 + vtcon unbind + `fb0/blank`=4 → `dpms`=Off 인데 모니터는 콘솔 유지. `status`=`off` (Weston 정지 상태, dmesg 무출력) 와 `i.mx8-hdp` unbind 는 모니터 켜짐/꺼짐 반복 (HDMI 5V 유지 → 입력 탐색 추정) | 소프트웨어로 모니터 대기 진입 불가. DDC 는 HDMI DDC 가 i2c 로 노출 안 됨 |
| `i.mx8-hdp` unbind → bind | bind 에서 커널 멈춤, SSH·ping 무응답. watchdog 꺼져 있어 자동 복구 없음 | 쓰지 말 것. 전원 재투입 |
| 절전 중 지난 `minutely` | 복귀 직후 한 번 실행됨 | 복귀 시각을 08:00 이후로 (07:59 복귀면 즉시 재절전) |
| 팬 소음 | `gpio-fan` on/off 2단, 기본 65°C (유휴 64°C 경계) | active trip 70°C (hyst 10°C), 야간 절전 |
| RTC 배터리 없음 | 전원 차단 후 `rtcwake -m off` 복귀 미검증 | `mem` 만 사용 (`snvs_rtc` 알람, deep) |
| 전원 재투입 후 RTC 1970 | RTC 1970 상태에서 `rtcwake -m mem` → 복귀 안 함 (전원 재투입 필요). NTP 동기화 후 커널 `RTC_SYSTOHC` 가 RTC 를 맞추면 정상 복귀 | NTP 동기화가 전제 |
| timesyncd 부팅 시 정지 | Wi-Fi 전에 기동 → `Packet count: 0` 으로 재시도 안 함 (NetworkManager 라 networkd 링크 신호 없음) | `wlan0` up 시 dispatcher 가 timesyncd 재시작 |
| 야간 TargetDown | 절전 중 `up{job="coral"}`=0 | AlertmanagerConfig `discord` 하위 route `muteTimeIntervals: coral-night` |
| 하드웨어 watchdog + 절전 | `imx2-wdt` 는 절전 중 최대 128초만 유지 → 128초 넘는 절전은 재부팅 (120초 시험은 통과, 9시간 절전은 2분 뒤 재부팅) | systemd `RuntimeWatchdogSec=0`. 멈춤은 `TargetDown` 으로 감지 |
| 모니터 전원 재투입 | HDMI Plug Out/In → Weston 출력 재생성, fullscreen 창 안 보임 | udev `HOTPLUG=1` → 3초 뒤 F11 두 번 (Chrome 재시작 없음) |
| `card0-HDMI-A-1/status` 로 hotplug 흉내 | Weston `ABRT` | 시험은 실제 모니터 전원으로. Weston `Restart=on-failure` |
| root 로 `xdotool` | `No protocol specified` (Xwayland 는 `mendel` 만 허용) | `runuser -u mendel` |
| 주사율 미지정 | `mode=1920x1080` → 목록 첫 번째 1080p120 (297MHz) | `1920x1080@60`. 모니터(PAQ2718) EDID 범위 48~144Hz 라 30Hz 불가, QHD 는 드라이버 클럭 표에 없음 |

수동 전환: `sudo kiosk-screen on|off|sleep|auto` — 다음 1분 타이머가 `auto` 로 덮어쓴다. 오래 유지하려면 `sudo systemctl stop kiosk-screen.timer`

### lifehub 키

`/etc/coral-kiosk/env` = `KIOSK_URL=https://health.json-server.win/kiosk?key=<KIOSK_KEY>`. 키는 git 에 없다.

| 값 | 위치 |
| --- | --- |
| `KIOSK_KEY` 원본 | WSL `~/.local/share/homelab/lifehub-kiosk.env` |
| 클러스터 | `k8s/health-hub/manifests/sealed-secret-kiosk.yaml` → `health-hub-dashboard` env |

키 교체: `KUBECONFIG=~/.kube/config-json kubeseal --raw --namespace health-hub --name lifehub-kiosk --scope strict` 로 봉인 → SealedSecret 갱신·머지 → `/etc/coral-kiosk/env` 수정 → `systemctl restart coral-kiosk`.

원격 화면: `ssh -fN -L 127.0.0.1:9222:127.0.0.1:9222 mendel@192.168.0.44` → Chrome `chrome://inspect` → Configure `127.0.0.1:9222`.

### 관측성 (mocha-snail)

| 레포 `nodes/kiosk/mocha-snail/` | 노드 경로 | 반영 |
| --- | --- | --- |
| `sysusers-node-exporter.conf` | `/etc/sysusers.d/node-exporter.conf` | `systemd-sysusers` |
| `node-exporter.service` | `/etc/systemd/system/node-exporter.service` | `systemctl daemon-reload && systemctl enable --now node-exporter` |
| `rsyslog-forward.conf` | `/etc/rsyslog.d/90-forward.conf` | `systemctl restart rsyslog` |

| 레포 밖 | 위치 | 생성 |
| --- | --- | --- |
| `node_exporter` v1.12.1 | `/usr/local/bin/node_exporter` | GitHub release `node_exporter-1.12.1.linux-arm64.tar.gz`, `sha256sums.txt` 검증 |

| 신호 | 경로 | 클러스터 쪽 |
| --- | --- | --- |
| 메트릭 | `192.168.0.44:9100` | `k8s/observability/prometheus/values.yaml` `job_name: coral` |
| 로그 | rsyslog → `192.168.0.27:5515/udp` | `k8s/observability/alloy-gateway/` `loki.source.syslog "coral"`, `syslog-lan-service.yaml` |

| 함정 | 증상 | 대응 |
| --- | --- | --- |
| `DynamicUser=yes` | systemd 수집기 `couldn't get dbus connection: ... broken pipe` | sysusers 고정 사용자 |
