#!/bin/bash
# Phase 4: hostPath PV 경로 변경 (/mnt/data → /mnt/hdd-1t) + 서비스 재시작
# Phase 3 완료 후 실행
set -euo pipefail

echo "=== [1/5] 기존 PV 삭제 (PVC는 유지됨) ==="
# PV spec.hostPath는 immutable → 삭제 후 재생성 필요
# PVC는 삭제하지 않음 — Lost 상태가 되지만 새 PV가 연결되면 복구됨
kubectl delete pv immich-library frigate-media-hdd seafile-data
echo "✅ PV 삭제 완료"

echo ""
echo "=== [2/5] 새 PV 생성 (/mnt/hdd-1t 경로로) ==="
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: PersistentVolume
metadata:
  name: immich-library
spec:
  capacity:
    storage: 1200Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  hostPath:
    path: /mnt/hdd-1t/immich-library
  nodeAffinity:
    required:
      nodeSelectorTerms:
        - matchExpressions:
            - key: kubernetes.io/hostname
              operator: In
              values:
                - json-server-1
  claimRef:
    namespace: immich
    name: immich-library
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: frigate-media-hdd
spec:
  capacity:
    storage: 200Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  hostPath:
    path: /mnt/hdd-1t/frigate-media
  nodeAffinity:
    required:
      nodeSelectorTerms:
        - matchExpressions:
            - key: kubernetes.io/hostname
              operator: In
              values:
                - json-server-1
  claimRef:
    namespace: frigate
    name: frigate-media-hdd
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: seafile-data
spec:
  capacity:
    storage: 100Gi
  storageClassName: hostpath-hdd
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  hostPath:
    path: /mnt/hdd-1t/seafile-data
  nodeAffinity:
    required:
      nodeSelectorTerms:
        - matchExpressions:
            - key: kubernetes.io/hostname
              operator: In
              values:
                - json-server-1
  claimRef:
    namespace: seafile
    name: seafile-data
EOF
echo "✅ PV 재생성 완료"

echo ""
echo "=== [3/5] PVC 상태 확인 ==="
kubectl get pvc -n immich immich-library
kubectl get pvc -n frigate frigate-media-hdd
kubectl get pvc -n seafile seafile-data

echo ""
echo "=== [4/5] 서비스 재시작 ==="
kubectl scale deployment -n immich immich-server immich-machine-learning --replicas=1
kubectl scale deployment -n seafile seafile --replicas=1
kubectl scale deployment -n frigate frigate --replicas=1
echo "✅ 서비스 재시작 완료"

echo ""
echo "=== [5/5] 파드 상태 확인 (30초 대기) ==="
sleep 30
kubectl get pods -n immich
kubectl get pods -n seafile
kubectl get pods -n frigate

echo ""
echo "=== ✅ Phase 4 완료 ==="
