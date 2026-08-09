# 노드 디스크 마운트 런북 (json-server-1)

hostPath PV가 올라탄 물리 디스크의 마운트를 보장하기 위한 노드 설정. `/etc/fstab`은 Git으로 관리되지 않으므로 여기에 목표 상태를 기록한다.

## 대상

| 디스크 | 마운트 지점 | 사용처 |
|---|---|---|
| `/dev/sdb1` (Seagate 1TB, LABEL `data-1tb`) | `/mnt/hdd-seagate-1t` | immich-library, seafile-data, frigate-media |
| `/dev/sdc1` (Samsung 640GB) | `/mnt/hdd-samsung-640g` | `local-path-hdd-samsung` StorageClass (Loki/Tempo/Grafana) |

## fstab 목표 상태

```
UUID=0958b4ac-f603-4e3f-8777-a6338fd0910b  /mnt/hdd-samsung-640g  ext4  defaults  0  2
UUID=97968160-1c0b-42f4-a94e-6a88b4014ac7  /mnt/hdd-seagate-1t    ext4  defaults,nofail,x-systemd.before=local-fs.target  0  2
```

### `x-systemd.before=local-fs.target`이 필요한 이유

`nofail`은 systemd-fstab-generator가 마운트 유닛에서 `Before=local-fs.target` 순서 제약을 **빼도록** 만든다. 그 결과 fsck가 아직 돌고 있어도 부팅이 `local-fs.target`을 통과하고, 뒤늦게 마운트 job이 취소되면서 fsck가 SIGTERM으로 잘린다. 마운트 유닛은 실패가 아니라 **취소**로 끝나므로 `Dependency failed` 로그조차 남지 않는다.

`x-systemd.before=`로 순서 제약만 되돌리면:

- 부팅이 fsck 완료까지 기다린다 (저널 재생은 보통 수 초)
- `Wants=`는 그대로라 디스크가 정말 죽어도 emergency mode로 빠지지 않고 부팅은 진행된다

`x-systemd.device-timeout=`은 이 문제에 쓰지 않는다. 그건 **디바이스 노드가 나타날 때까지**의 대기 시간이지 fsck 대기와 무관하다.

적용:

```bash
sudo cp /etc/fstab /etc/fstab.bak
sudo sed -i 's|\(UUID=97968160-1c0b-42f4-a94e-6a88b4014ac7.*\)defaults,nofail|\1defaults,nofail,x-systemd.before=local-fs.target|' /etc/fstab
sudo systemctl daemon-reload
systemctl cat 'mnt-hdd\x2dseagate\x2d1t.mount' | grep -E 'Before|Options'   # Before=local-fs.target 확인
```

## 마운트 지점 스텁 차단

hostPath PV에 `type: Directory`를 걸어도, 마운트 지점 **아래 루트 파일시스템에 빈 디렉토리가 남아 있으면** kubelet 검사를 통과해버린다. 컨테이너 런타임은 bind-mount 원본이 없으면 자동 생성하므로, 디스크가 안 붙은 채 파드가 뜨면 스텁이 생긴다.

마운트된 상태에서는 스텁이 가려져 보이지 않는다. bind-mount로 루트 파일시스템을 직접 확인한다:

```bash
sudo mkdir -p /tmp/rootbind && sudo mount --bind / /tmp/rootbind
ls -la /tmp/rootbind/mnt/hdd-seagate-1t/          # 비어 있어야 정상
```

스텁이 있으면 제거하고 마운트 지점 자체를 잠근다:

```bash
sudo rm -rf /tmp/rootbind/mnt/hdd-seagate-1t/*
sudo chattr +i /tmp/rootbind/mnt/hdd-seagate-1t   # 하위 생성 차단 (마운트는 정상 동작)
sudo umount /tmp/rootbind && sudo rmdir /tmp/rootbind
```

`chattr +i`는 디렉토리 위에 파일시스템을 마운트하는 것은 막지 않는다. 마운트가 없을 때만 쓰기를 차단한다.

## 미마운트 복구

```bash
sudo e2fsck -n -f /dev/sdb1        # 1. 읽기 전용 검사. bitmap 차이만 있고 Pass 2~5가 조용하면 저널 재생으로 충분
sudo e2fsck -p /dev/sdb1           # 2. 안전한 항목만 자동 수정 (애매하면 스스로 중단)
sudo mount /mnt/hdd-seagate-1t
kubectl -n immich rollout restart deploy/immich-server
kubectl -n seafile rollout restart deploy/seafile
```

`-n` 결과 판독: 모든 차이가 `-`(비트맵은 사용 중이라는데 실제로는 빈 블록) 방향이면 반영 안 된 free 공간 회계이므로 안전하다. `+` 방향이 섞여 있거나 Pass 1에서 `illegal block`·`multiply-claimed block`이 나오면 실제 손상이므로 진행 전 백업 상태를 먼저 확인한다.

## 감시

- `HostPathDiskUnmounted` — `/mnt/hdd-seagate-1t` 미마운트 5분 지속 시 critical
- `NodeRebooted` — 노드 부팅 후 10분 이내 warning

정의는 [alert-rules.yaml](../k8s/observability/prometheus/manifests/alert-rules.yaml)의 `storage` / `node` 그룹.
