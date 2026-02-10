#!/bin/bash
# MinIO credentials SealedSecret 생성 스크립트
# 클러스터 접근이 가능할 때 실행하세요.
#
# 사전 준비:
# 1. kubectl, kubeseal 설치 및 클러스터 접근 가능
# 2. factorio namespace 존재

set -euo pipefail

MINIO_USER="minioadmin"
MINIO_PASS=$(openssl rand -base64 24)

echo "MinIO Root User: $MINIO_USER"
echo "MinIO Root Password: $MINIO_PASS"

kubectl create secret generic minio-credentials \
  --namespace=factorio \
  --from-literal=rootUser="$MINIO_USER" \
  --from-literal=rootPassword="$MINIO_PASS" \
  --dry-run=client -o yaml \
  | kubeseal --format yaml \
    --merge-into sealed-secret.yaml 2>/dev/null \
  || kubectl create secret generic minio-credentials \
    --namespace=factorio \
    --from-literal=rootUser="$MINIO_USER" \
    --from-literal=rootPassword="$MINIO_PASS" \
    --dry-run=client -o yaml \
    | kubeseal --format yaml > sealed-secret.yaml

# Reflector 어노테이션은 sealed-secret.yaml의 template.metadata에 이미 포함
echo "Created: sealed-secret.yaml"
echo "Don't forget to commit and push!"
