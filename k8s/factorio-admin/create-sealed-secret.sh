#!/bin/bash
# factorio-admin-secrets SealedSecret 생성 스크립트
# 클러스터 접근이 가능할 때 실행하세요.
#
# 사전 준비:
# 1. GitHub OAuth App 생성 후 Client ID / Secret 획득
#    Callback URL: https://factorio-admin.json-server.win/api/auth/callback/github
# 2. kubectl, kubeseal 설치 및 클러스터 접근 가능

set -euo pipefail

# 값 입력
read -p "GitHub Client ID: " GITHUB_CLIENT_ID
read -sp "GitHub Client Secret: " GITHUB_CLIENT_SECRET; echo
read -sp "RCON Password (Factorio 서버와 동일해야 함): " RCON_PASSWORD; echo
POSTGRES_PASSWORD=$(openssl rand -base64 24)
AUTH_SECRET=$(openssl rand -base64 32)

echo "Generated Postgres password: $POSTGRES_PASSWORD"
echo "Generated Auth secret: $AUTH_SECRET"

# Secret YAML 생성 (dry-run)
kubectl create secret generic factorio-admin-secrets \
  --namespace=factorio \
  --from-literal=auth-secret="$AUTH_SECRET" \
  --from-literal=github-client-id="$GITHUB_CLIENT_ID" \
  --from-literal=github-client-secret="$GITHUB_CLIENT_SECRET" \
  --from-literal=rcon-password="$RCON_PASSWORD" \
  --from-literal=postgres-password="$POSTGRES_PASSWORD" \
  --dry-run=client -o yaml | kubeseal --format yaml > sealed-secret.yaml

echo "Created: sealed-secret.yaml"
echo "Don't forget to commit and push!"

# 원본은 메모리에만 존재하므로 안전합니다
