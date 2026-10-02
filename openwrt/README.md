# OpenWrt (ipTIME AX3000SM)

집 공유기 설정. OpenWrt 25.12.5 `mediatek/filogic` 타겟, 보드 `iptime_ax3000sm`.

`config/` 는 기기의 `/etc/config/` 를 그대로 받은 것이다. **무선 비밀번호만 `REPLACE_ME` 플레이스홀더**이고 나머지는 실제 값이다.

## 현재 설정

| 항목 | 값 |
| --- | --- |
| hostname | `ax3000sm` |
| 시간대 | `Asia/Seoul` (KST-9) |
| LAN | `192.168.0.1/24` |
| WAN | `eth1`, DHCP |
| 무선 | `5G` (5GHz, 채널 48) / `2.4G` (2.4GHz, 자동), WPA2-PSK + CCMP, 국가 KR |
| DHCP 임대 | 24시간, 풀 `.100` ~ `.249` |
| DHCP 예약 | `.24` galaxybook2 / `.27` json-server-1 / `.37` raspi-1 / `.44` mocha-snail / `.46` json-server-2 |
| 포트포워딩 | TCP 80, 443 → `192.168.0.27` |

2283(Immich LAN 평문 엔드포인트)은 포워딩하지 않는다.

UPnP 는 `miniupnpd` 가 미설치라 동작하지 않는다. WAN 쪽 관리 접근은 방화벽 기본값 `wan: input REJECT` 로 차단된다.

## 메트릭

`prometheus-node-exporter-lua` 가 `192.168.0.1:9100/metrics` 를 LAN 에만 바인딩해 제공한다 (`listen_interface lan`). Prometheus 가 static 대상으로 긁는다 — `k8s/observability/prometheus/values.yaml` 의 `additionalScrapeConfigs`.

설치 패키지는 `packages-extra.txt` 에 있다.

```bash
apk add $(cat openwrt/packages-extra.txt | tr '\n' ' ')
```

진단에 쓰는 지표:

| 지표 | 쓰임 |
| --- | --- |
| `node_ethtool_rxpause` / `txpause` (device=lan1~4) | PAUSE 프레임. 공유기가 LAN 을 멈추게 하는 순간을 공유기 쪽에서 직접 본다 |
| `node_network_receive_bytes_total` (device=phy1-ap0, lan*) | 무선→유선 중계량 |
| `node_cpu_seconds_total` | CPU 포화 여부. WED 오프로드 필요성 판단 근거 |
| `node_thermal_zone_temp` | SoC 온도, 스로틀링 감지 |
| `node_nf_conntrack_entries` / `_limit` | 연결 추적 테이블 포화 |

클라이언트별 신호·전송률은 장치 단위로만 노출돼(값 0) 쓸 수 없다.

## 로그

공유기 로그는 기본적으로 RAM 링버퍼(`logd -S 128`)에만 있어 재부팅하면 사라진다. 내장 원격 전송으로 클러스터 otel-collector 에 보낸다.

```text
system.@system[0].log_ip   = 192.168.0.27
system.@system[0].log_port = 5514
system.@system[0].log_proto = udp
```

**새 데몬을 올리지 않는다.** 이미 돌고 있는 `logd` 가 UDP 로 한 번 더 쓰는 것이라 공유기 부하가 측정되지 않는 수준이다 (적용 전후 메모리·프로세스 수 동일).

받는 쪽은 otel-collector 의 `syslog` 수신기다 (`protocol: rfc3164` — OpenWrt `logd` 는 구형 BSD 형식으로 보낸다). 로그는 `service.name=openwrt` 라벨로 Loki 에 들어간다.

hostapd(접속·인증·DFS), dnsmasq(DHCP·DNS), 커널(mt76·링크 이벤트)이 여기로 흐른다.

## 설정 복원

```bash
scp -i ~/.ssh/openwrt_ax3000sm openwrt/config/* root@192.168.0.1:/etc/config/
ssh -i ~/.ssh/openwrt_ax3000sm root@192.168.0.1 \
  "uci set wireless.default_radio0.key='<무선비밀번호>'; \
   uci set wireless.default_radio1.key='<무선비밀번호>'; \
   uci commit; reload_config"
```

`network` 의 `dhcp_default_duid` 와 `ula_prefix` 는 기기가 생성한 값이다. 그대로 복원하면 IPv6 주소가 유지된다.

## 펌웨어 설치

ipTIME 복구 부트로더에 TFTP 로 `factory` 이미지를 넣는다. 시리얼 불필요.

1. PC 유선 인터페이스를 `192.168.0.100/24` 고정으로 설정
2. 공유기 리셋 버튼을 누른 채 전원 투입, CPU LED 깜빡임이 멈출 때까지 10초 이상 유지
3. 전송

```bash
curl --tftp-no-options -T openwrt-*-iptime_ax3000sm-squashfs-factory.bin \
  tftp://192.168.0.1/ipTIME_FIRM_WARE
```

파일 이름은 `ipTIME_FIRM_WARE` 여야 부트로더가 펌웨어로 인식한다.

윈도우에서는 **방화벽 인바운드 허용이 필요하다.** TFTP 서버는 69번이 아닌 새 포트로 응답해서 상태 기반 방화벽이 차단한다 — ping 은 되는데 전송만 0바이트인 증상으로 나타난다.

```powershell
New-NetFirewallRule -DisplayName "TFTP-recovery" -Direction Inbound -Protocol UDP -RemoteAddress 192.168.0.1 -Action Allow
```

순정 복귀도 같은 절차에 ipTIME 순정 `.bin` 을 넣으면 된다.

## 미적용

**WED (Wireless Ethernet Dispatch) 하드웨어 오프로드.** 커널은 `CONFIG_NET_MEDIATEK_SOC_WED=y` 로 빌드돼 있으나 mt76 드라이버 기본값이 꺼짐이다. `/etc/modules.d/` 에 `mt7915e wed_enable=1` 이 필요하고, 적용 후 `dmesg | grep -i wed` 와 대용량 무선 업로드로 검증한다.
