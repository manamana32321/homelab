# Reflector

Secret과 ConfigMap을 네임스페이스 간에 복제합니다.

## 사용 방법

### 1. 소스 Secret에 annotation 추가

복제를 허용할 Secret에 다음 annotation을 추가합니다:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-secret
  namespace: source-namespace
  annotations:
    reflector.v1.k8s.emberstack.com/reflection-allowed: "true"
type: Opaque
```

### 2. 대상 namespace에 reflected Secret 생성

복제할 namespace에 빈 Secret을 만들고 `reflects` annotation을 추가합니다:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-secret
  namespace: target-namespace
  annotations:
    reflector.v1.k8s.emberstack.com/reflects: "source-namespace/my-secret"
type: Opaque
```

### 3. 자동 복제 (선택)

특정 namespace로 자동 복제하려면 소스에 다음 annotation을 추가합니다:

```yaml
annotations:
  reflector.v1.k8s.emberstack.com/reflection-allowed: "true"
  reflector.v1.k8s.emberstack.com/reflection-auto-enabled: "true"
  reflector.v1.k8s.emberstack.com/reflection-auto-namespaces: "namespace1,namespace2"
```

## 현재 복제 중인 Secret

| 소스 | 대상 namespace | 용도 |
|------|----------------|------|
| `cloudflare-credentials/cloudflare-api-token` | cert-manager | DNS01 인증서 발급 |

## 상태 확인

```bash
# 복제된 Secret 확인
kubectl get secrets -A -l reflector.v1.k8s.emberstack.com/reflects

# Reflector 로그
kubectl logs -n kube-system -l app.kubernetes.io/name=reflector
```

## ConfigMap 복제

Secret과 동일한 방식으로 ConfigMap도 복제할 수 있습니다:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: source-namespace
  annotations:
    reflector.v1.k8s.emberstack.com/reflection-allowed: "true"
```
