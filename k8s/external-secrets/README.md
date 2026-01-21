# External Secrets Operator

AWS Secrets Manager의 시크릿을 Kubernetes Secret으로 동기화합니다.

## 사용 방법

### 1. ClusterSecretStore 설정

AWS credentials를 사용하는 ClusterSecretStore를 먼저 생성합니다:

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: aws-secrets-manager
spec:
  provider:
    aws:
      service: SecretsManager
      region: ap-northeast-2
      auth:
        secretRef:
          accessKeyIDSecretRef:
            name: aws-credentials
            namespace: external-secrets
            key: aws_access_key_id
          secretAccessKeySecretRef:
            name: aws-credentials
            namespace: external-secrets
            key: aws_secret_access_key
```

### 2. ExternalSecret 생성

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: my-app-secrets
  namespace: <YOUR_NAMESPACE>
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: ClusterSecretStore
  target:
    name: my-app-secrets
    creationPolicy: Owner
  data:
    - secretKey: DATABASE_URL
      remoteRef:
        key: my-app/database
        property: url
```

### 3. 전체 JSON 시크릿 가져오기

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: my-app-secrets
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: ClusterSecretStore
  target:
    name: my-app-secrets
  dataFrom:
    - extract:
        key: my-app/config
```

### 4. 상태 확인

```bash
# ExternalSecret 상태
kubectl get externalsecret -n <namespace>

# 동기화 상태 상세
kubectl describe externalsecret <name> -n <namespace>

# 생성된 Secret 확인
kubectl get secret <target-name> -n <namespace> -o yaml

# ClusterSecretStore 상태 확인
kubectl get clustersecretstore

# External Secrets Operator 로그
kubectl logs -n external-secrets -l app.kubernetes.io/name=external-secrets
```

## AWS Credentials 설정

AWS Secrets Manager 접근을 위해 AWS credentials가 필요합니다.
SealedSecret + Reflector 패턴으로 external-secrets namespace에 복제합니다.
