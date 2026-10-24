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

HDMI 모니터에 GNOME 자동 로그인(`kiosk`) → Chrome 전체화면 Grafana. RDP 로 같은 화면 공유. 08:00 켜짐, 23:00 꺼짐.

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

수동 전환: `$U kiosk-screen on|off|auto`
