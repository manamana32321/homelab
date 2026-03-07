#!/bin/bash
# Phase 3: mergerfs 해제
# rsync 완료 확인 후 실행할 것!
set -euo pipefail

SERVER1="192.168.0.27"

echo "=== [1/4] rsync 완료 확인 ==="
RSYNC_COUNT=$(ssh $SERVER1 "ps aux | grep '[r]sync -a /mnt/hdd' | wc -l")
if [ "$RSYNC_COUNT" -gt "0" ]; then
  echo "❌ rsync 아직 실행 중! 완료 후 다시 실행하세요."
  exit 1
fi
echo "✅ rsync 완료 확인"

echo ""
echo "=== [2/4] 데이터 검증 (1TB 주요 디렉토리 크기 확인) ==="
ssh $SERVER1 "du -sh /mnt/hdd-1t/*"

echo ""
echo "=== [3/4] mergerfs 언마운트 ==="
ssh $SERVER1 "sudo umount /mnt/data && echo '✅ mergerfs 언마운트 완료'"

echo ""
echo "=== [4/4] fstab 업데이트 ==="
ssh $SERVER1 "
  sudo cp /etc/fstab /etc/fstab.bak-\$(date +%Y%m%d)
  # mergerfs 라인 제거
  sudo sed -i '/fuse.mergerfs/d' /etc/fstab
  # hdd-seagate 관련 라인 제거
  sudo sed -i '/hdd-seagate/d' /etc/fstab
  echo '✅ fstab 업데이트 완료'
  echo ''
  echo '--- 현재 fstab ---'
  cat /etc/fstab
"

echo ""
echo "=== ✅ Phase 3 완료 ==="
echo "다음: phase4-update-pvs.sh 실행"
