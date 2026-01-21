# Cert Manager

Let's Encrypt 인증서를 자동으로 발급하고 관리합니다.

## Ingress에 HTTPS 적용

### 1. Ingress에 annotation 추가

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-app
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-dns01
spec:
  ingressClassName: traefik
  tls:
    - hosts:
        - my-app.json-server.win
      secretName: my-app-tls
  rules:
    - host: my-app.json-server.win
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: my-app
                port:
                  number: 80
```

### 2. 지원 도메인

| 도메인 | DNS Provider |
|--------|--------------|
| `*.json-server.win` | Cloudflare |

### 3. 인증서 발급 확인

```bash
# Certificate 상태 확인
kubectl get certificate -n <namespace>

# 상세 정보
kubectl describe certificate <name> -n <namespace>

# 발급 과정 로그
kubectl logs -n cert-manager -l app=cert-manager
```

### 4. 트러블슈팅

인증서 발급이 안 될 경우:

```bash
# CertificateRequest 확인
kubectl get certificaterequest -n <namespace>

# Challenge 상태 확인 (DNS 검증)
kubectl get challenge -A

# Order 확인
kubectl get order -n <namespace>
```

## Cloudflare Credentials

Cloudflare DNS01 검증을 위해 `cloudflare-credentials` namespace의 Secret을 Reflector로 복사해서 사용합니다.

- `cloudflare-api-token` → json-server.win
