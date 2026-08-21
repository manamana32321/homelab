# NIC TX 큐 행 자동 복구 런북 (json-server-1)

## 배경

2026-08-21, json-server-1 의 온보드 Realtek NIC(`enp3s0`, r8169)이 송신 큐를 물고 놓지 않는
상태에 빠졌다. 커널 netdev watchdog 이 5초마다 이를 고발했지만 드라이버 자체 리셋은 계속 실패했다.

```
r8169 0000:03:00.0 enp3s0: NETDEV WATCHDOG: CPU: 7: transmit queue 0 timed out 5567 ms
r8169 0000:03:00.0 enp3s0: rtl_txcfg_empty_cond == 0 (loop: 666, delay: 100).
```

21분간 153회 반복, 자력 복구 0회. 노드는 살아있었지만 송신이 막혀 SSH·apiserver 모두 응답
불가였고, 결국 손으로 전원을 끊어야 했다. json-server-1 은 단일 control-plane 이므로
NIC 하나가 클러스터 API 전체를 멈춰 세웠다.

이 런북은 그 상황을 사람 없이 수초 안에 빠져나오기 위한 호스트 레벨 유닛을 다룬다.

## 동작

`hosts/json-server-1/nic-tx-watchdog.sh` 가 커널 저널을 따라가다 해당 인터페이스의
`NETDEV WATCHDOG` 라인을 만나면 3단계로 올라간다.

| 단계 | 동작 | 부작용 | 판정 |
|---|---|---|---|
| 1 | `ip link set enp3s0 down/up` | 없음 | 20초 내 회복 여부 |
| 2 | `modprobe -r/-a r8169` + `systemctl restart k3s` | API 30초 블립 | 60초 내 회복 여부 |
| 3 | `systemctl reboot` | 노드 재부팅 | 누적 실패 300초 초과 시 |

**1단계가 먼저인 이유**: `ip -d link show flannel.1` 을 보면 flannel VXLAN 이
`vxlan id 1 local 192.168.0.27 dev enp3s0` 로 물리 NIC 을 언더레이로 직접 참조한다.
모듈을 내리면 netdev 가 사라져 터널도 같이 무너지므로, NIC 이 살아 돌아와도 파드 네트워킹은
죽은 채 남는다. 링크 바운스는 netdev 를 유지한 채 칩만 재초기화하므로 이 문제가 없다.
2단계로 내려갈 때 k3s 를 재시작하는 것도 같은 이유다.

**3단계가 재부팅인 이유**: 실패 시 대안이 "손으로 전원 끊기"였다. clean reboot 은 Longhorn
볼륨을 정상 언마운트하고 kine SQLite 가 체크포인트를 남긴다. 루트가 DRAM-less SSD 라
이 차이가 크다. `/run` 마커로 **부팅당 1회**만 허용되므로 재부팅 루프에 빠지지 않는다.

### 오탐 방지

- 트리거는 게이트웨이 ping 실패가 아니라 **커널의 `NETDEV WATCHDOG` 라인**이다. 공유기가
  죽어도 이 라인은 안 나오므로 공유기 장애로는 복구 동작이 돌지 않는다.
- 회복 판정은 캐리어 up + (게이트웨이 응답 **또는** TX 카운터 전진). 공유기가 같이 죽은
  상황에서 NIC 만 살아난 경우를 실패로 오판하지 않기 위함이다.
- 행 중에는 watchdog 라인이 ~5초마다 쏟아지므로 복구 사이클 간 `COOLDOWN_SEC`(기본 120초)
  간격을 둔다.

## 설치

```bash
sudo install -m 0755 hosts/json-server-1/nic-tx-watchdog.sh /usr/local/sbin/nic-tx-watchdog.sh
sudo install -m 0644 hosts/json-server-1/nic-tx-watchdog.service /etc/systemd/system/nic-tx-watchdog.service
sudo systemctl daemon-reload
sudo systemctl enable --now nic-tx-watchdog.service
```

## 검증

실제 복구를 돌리지 않고 매칭 로직·연결 상태만 점검한다.

```bash
sudo /usr/local/sbin/nic-tx-watchdog.sh --self-test
```

전부 `OK` 여야 한다. 서비스 상태와 시작 로그:

```bash
systemctl status nic-tx-watchdog.service
journalctl -u nic-tx-watchdog.service -n 20
```

### 실제 복구 경로까지 확인하고 싶을 때

1단계(링크 바운스)를 손으로 재현하면 된다. **enp3s0 이 수초간 끊기므로 apiserver 와
Longhorn 에 블립이 생긴다.** 한가한 시간에만 할 것.

```bash
sudo ip link set dev enp3s0 down; sleep 3; sudo ip link set dev enp3s0 up
```

## 튜닝

`/etc/systemd/system/nic-tx-watchdog.service` 의 `Environment=` 로 조정한다.

| 변수 | 기본값 | 의미 |
|---|---|---|
| `IFACE` | `enp3s0` | 감시 대상 인터페이스 |
| `MODULE` | `r8169` | 2단계에서 재적재할 드라이버 |
| `GATEWAY` | `192.168.0.1` | 회복 판정용 ping 대상 |
| `COOLDOWN_SEC` | `120` | 복구 사이클 간 최소 간격 |
| `REBOOT_AFTER_SEC` | `300` | 누적 실패가 이 시간을 넘으면 재부팅. **`0` 이면 재부팅 폴백 비활성** |

변경 후 `sudo systemctl daemon-reload && sudo systemctl restart nic-tx-watchdog.service`.

## 제거

```bash
sudo systemctl disable --now nic-tx-watchdog.service
sudo rm /etc/systemd/system/nic-tx-watchdog.service /usr/local/sbin/nic-tx-watchdog.sh
sudo systemctl daemon-reload
```

## 이건 임시방편이다

근본 원인은 2011년 엔트리 칩셋(ASUS P8H61-M) 온보드 Realtek 컨트롤러 + 2012년 펌웨어
(`rtl8168e-3_0.0.4`)다. 흔한 회피책(오프로드 끄기, ASPM 끄기)은 이미 적용된 상태라 남은
근본 해결은 둘뿐이다.

- NIC 물리 교체 (인텔 PCIe NIC 또는 USB3 GbE 어댑터)
- control-plane 을 json-server-2 로 이전 — 단, 그 전에 j2 의 `eno1` 이 100 Mb/s 로 링크되는
  문제를 먼저 잡아야 한다

재발 이력은 `journalctl -u nic-tx-watchdog.service | grep '복구 시작'` 으로 센다.
자주 뜨기 시작하면 임시방편의 수명이 끝난 것이다.
