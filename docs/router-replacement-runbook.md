# 공유기 교체 Runbook (ipTIME A1004 → AX3000SM)

이 문서는 **순서대로** 실행한다. Phase 2(물리 교체) 구간에는 개발 머신도 네트워크가 끊기므로 사람이 직접 수행한다.

## 왜 교체하는가

A1004(MediaTek MT7620A, 싱글코어 580MHz, RAM 64MB)는 5GHz 무선을 유선으로 넘기는 작업을 CPU가 소프트웨어로 처리한다. 폰에서 사진/영상을 80~130Mbps로 1~2분 올리면 이 CPU가 한계를 넘고, 내장 스위치가 모든 포트에 PAUSE 프레임을 뿌려 **LAN 전체(서버 간 통신 포함)가 멈춘다**. 재부팅해야만 풀린다.

2026-09-19 하루에 두 번 발생했고, 07-10·08-21·09-05 인시던트도 같은 패턴이었다. 상세 분석은 `project-incident-2026-09-19-router-pause-storm` 참조.

AX3000SM(MT7981BA, 듀얼코어 A53 1.3GHz, RAM 256MB)은 무선↔유선 전달을 하드웨어(WED)로 처리한다.

## 전제 조건

- [ ] AX3000SM 수령
- [ ] 새 무선 비밀번호 결정 (SSID는 **기존과 동일**하게 유지 — Immich 앱의 SSID 기반 자동 URL 전환이 여기 의존)
- [ ] 기존 공유기 포트포워딩 목록 스크린샷 (80/443 외 항목 확인용)
- [ ] 백업 CronJob 7종 최근 성공 확인

```bash
kubectl --context json get cronjob -A -o custom-columns='NS:.metadata.namespace,NAME:.metadata.name,LAST:.status.lastSuccessfulTime' | grep -iE 'backup|NS'
```

## 참조값

| 노드 | IP | 인터페이스 | MAC |
|---|---|---|---|
| json-server-1 | 192.168.0.27 | enp3s0 | `bc:ae:c5:70:d1:83` |
| json-server-2 | 192.168.0.46 | eno1 | `70:85:c2:67:78:b2` |
| raspi-1 | 192.168.0.37 | eth0 | `d8:3a:dd:c1:68:25` |

내부 대역 `192.168.0.0/24`, 게이트웨이 `192.168.0.1`. 이 값들은 kubeconfig(`https://192.168.0.27:6443`), k3s 에이전트 서버 주소, apiserver 인증서 SAN, Tailscale 광고 경로, blackbox probe 타겟, Immich 앱 로컬 주소에 박혀 있으므로 **바꾸지 않는다**.

## 1. 새 공유기 사전 설정 (교체 전, 노트북 직결)

- [ ] 내부 대역 `192.168.0.0/24`, 게이트웨이 `.1` 확인
- [ ] 무선 SSID 기존과 동일, 비밀번호는 새 값
- [ ] DHCP 예약 3건 (위 MAC 표)
- [ ] DHCP 임대 시간 **24시간 이상** (기본 2시간이면 공유기 장애 1시간에도 노드가 IP를 잃을 수 있음)
- [ ] 포트포워딩 **80, 443 → 192.168.0.27**
- [ ] **2283 포워딩 금지** — Immich LAN 전용 평문 엔드포인트다 (`k8s/immich/manifests/server-lan-service.yaml`)

## 2. 물리 교체 (사람)

- [ ] 랜선 라벨링: j1 / j2 / raspi / WAN
- [ ] 구형 공유기 분리 → 신형 연결 → 노드 3대 + WAN 결선
- [ ] **구형 A1004 보관** (롤백용)

## 3. 복구 확인

```bash
# 노드가 예약된 IP를 받았는지
kubectl --context json get nodes -o wide

# LAN 계층 probe 4개
kubectl --context json -n observability exec deploy/blackbox-exporter-prometheus-blackbox-exporter -- \
  sh -c 'for t in 192.168.0.1 192.168.0.27 192.168.0.46 192.168.0.37; do wget -qO- "localhost:9115/probe?target=$t&module=icmp_ping" | grep "^probe_success"; done'

# 볼륨 상태
kubectl --context json -n longhorn-system get volumes.longhorn.io \
  -o custom-columns='PVC:.status.kubernetesStatus.pvcName,STATE:.status.state,ROBUST:.status.robustness'
```

- [ ] 케이블 탈착으로 XFS 셧다운이 생겼을 수 있으므로 PVC 파드 I/O 스윕 (`ls <mountPath>`가 `Input/output error`면 해당 파드 재시작)
- [ ] 외부 접속 확인: `curl -I https://photos.json-server.win`
- [ ] 공인 IP 변동 확인. 바뀌었으면 `.envrc.local`의 `TF_VAR_default_ip` 갱신 후 `cd cloudflare && terraform apply`

```bash
curl -s https://api.ipify.org
```

## 4. 무선 비밀번호 반영

netplan에 무선 비밀번호가 평문으로 들어 있는 노드가 있다. 유선으로 붙어 있으므로 SSH로 수정해도 안전하다.

- [ ] json-server-1: `/etc/netplan/50-cloud-init.yaml`, 인터페이스 `wlxb0386cf0a9b5`
- [ ] raspi-1: `/etc/netplan/50-cloud-init.yaml`, 인터페이스 `wlan0`
- [ ] 폰·개발 머신 재연결

```bash
sudo netplan apply && nmcli -f DEVICE,STATE device status 2>/dev/null || networkctl status
```

## 5. 업데이트 + 재부팅 (한 대씩)

영향이 작은 순서로 진행하고, 각 단계마다 볼륨이 정상으로 돌아온 것을 확인한 뒤 다음으로 넘어간다.

1. **raspi-1** — 워크로드 적음
2. **json-server-2** — Longhorn replica 15개가 여기 있어 DB 대부분이 잠시 중단된다
3. **json-server-1** — control-plane. 재부팅 동안 Prometheus·Immich도 함께 중단된다

```bash
sudo apt update && sudo apt upgrade -y && sudo reboot
```

재부팅으로 볼륨이 faulted 되어도 `storageOverProvisioningPercentage: 120`(homelab#322) 덕분에 auto-salvage가 동작한다. 복구되지 않으면 `reference-longhorn-salvage-needs-schedulable-disk` 참조.

```bash
kubectl --context json get nodes
kubectl --context json get po -A --field-selector status.phase!=Running,status.phase!=Succeeded
```

## 6. 최종 검증

- [ ] 집 무선에서 **1GB 이상 영상 업로드**
- [ ] 업로드 중 probe 4개 유지, j2 PAUSE 카운터 증가 없음 → 원인 해소 확정

```bash
ssh json@192.168.0.46 'sudo ethtool -S eno1 | grep -E "rx_flow_control_xoff|tx_timeout"'
ssh json@192.168.0.46 'sudo journalctl -k --since "-30 min" | grep -E "Reset adapter|Link is Down"'
```

- [ ] Immich 앱이 로컬 엔드포인트(`http://192.168.0.27:2283`)를 사용하는지 — 업로드 후 traefik 액세스 로그에 `photos.json-server.win` 요청이 없으면 성공

## 롤백

구형 A1004를 다시 연결하면 된다. 노드는 DHCP라 별도 복구 작업이 없다. 단 무선 비밀번호를 이미 변경했다면 구형 공유기는 이전 비밀번호를 쓰므로 기기 재연결이 필요하다.

## 교체 후

- [ ] 이 문서를 실행 결과에 맞게 갱신하거나, 완료 후 삭제하고 `k8s/CLAUDE.md`의 네트워크 서술만 유지
- [ ] OpenWrt 전환을 검토한다면 별도 작업으로 분리한다 (`iptime_ax3000sm` 이미지는 OpenWrt 25.12 정식 릴리스에 포함, ipTIME 복구 프로그램으로 설치·복귀 가능)
