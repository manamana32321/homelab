#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# .env 파일에서 Telegram 설정 로드
if [ ! -f .env ]; then
  echo "Error: .env file not found. Copy .env.example to .env and fill in values."
  exit 1
fi
set -a
source .env
set +a

# alertmanager.yml 템플릿에서 실제 값 치환
envsubst '${TELEGRAM_BOT_TOKEN} ${TELEGRAM_CHAT_ID} ${TELEGRAM_MESSAGE_THREAD_ID}' < alertmanager.yml > alertmanager-rendered.yml

docker compose up -d

echo "Monitoring stack deployed."
echo "  Blackbox Exporter: http://$(hostname -I | awk '{print $1}'):9115"
echo "  Prometheus:        http://$(hostname -I | awk '{print $1}'):9090"
echo "  Alertmanager:      http://$(hostname -I | awk '{print $1}'):9093"
