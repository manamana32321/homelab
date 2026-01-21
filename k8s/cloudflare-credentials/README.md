# Cloudflare Credentials

cert-manager DNS01 검증용 Cloudflare API Token을 관리합니다.

## 설정 방법

### 1. Cloudflare API Token 생성

[Cloudflare API Tokens](https://dash.cloudflare.com/profile/api-tokens) 페이지에서:

1. "Create Token" 클릭
2. "Edit zone DNS" 템플릿 선택 또는 Custom Token 생성:
   - Permissions: Zone > DNS > Edit
   - Zone Resources: Include > Specific zone > json-server.win
3. Token 복사

### 2. SealedSecret 생성

```bash
# 일반 Secret 생성 (dry-run)
kubectl create secret generic cloudflare-api-token \
  --namespace=cloudflare-credentials \
  --from-literal=api-token=<YOUR_TOKEN> \
  --dry-run=client -o yaml > secret.yaml

# SealedSecret으로 암호화
kubeseal --format yaml < secret.yaml > cloudflare-api-token.yaml

# 원본 삭제
rm secret.yaml
```

### 3. cert-manager namespace로 복제

Reflector가 자동으로 cert-manager namespace로 복제합니다.

cert-manager namespace에 reflected Secret 생성:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: cloudflare-api-token
  namespace: cert-manager
  annotations:
    reflector.v1.k8s.emberstack.com/reflects: "cloudflare-credentials/cloudflare-api-token"
type: Opaque
```

## 사용처

| 컴포넌트 | 용도 |
|----------|------|
| cert-manager | Let's Encrypt DNS01 검증 (json-server.win) |
