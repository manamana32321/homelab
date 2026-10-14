# Cloudflare Credentials

cert-manager DNS01 검증용 Cloudflare API Token을 관리합니다.

## 설정 방법

### 1. Cloudflare API Token 발급 (Terraform)

토큰은 [cloudflare/tokens.tf](../../cloudflare/tokens.tf)의 `cloudflare_account_token.cert_manager`가 SSOT (Zone Read + DNS Write, json-server.win 한정).
대시보드에서 손으로 만들지 않는다. 토큰이 무효화되면 재발급:

```bash
cd cloudflare
terraform plan -replace=cloudflare_account_token.cert_manager   # 확인 후
terraform apply -replace=cloudflare_account_token.cert_manager
```

> 이력: 2026-10 기존 토큰(대시보드 수동 발급)이 Cloudflare에서 무효화돼 DNS01 발급·갱신이 전부 실패했다 (`Invalid API Token`). 그 뒤 Terraform 관리로 전환.

### 2. SealedSecret 생성

homelab 클러스터(`~/.kube/config-json`) cert로 sealing한다. 토큰 값은 화면에 출력하지 않고 파이프로만 넘긴다.

```bash
terraform -chdir=cloudflare output -raw cert_manager_token \
  | KUBECONFIG=~/.kube/config-json kubeseal --raw \
      --namespace=cloudflare-credentials --name=cloudflare-api-token
# 출력값을 cloudflare-api-token.yaml의 spec.encryptedData.api-token에 넣는다
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
