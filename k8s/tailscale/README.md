# Tailscale subnet router

홈랩 LAN(`192.168.0.0/24`)을 tailnet 에 광고해서, 외부 네트워크에서 kubectl·공유기
관리페이지·LAN 상의 모든 호스트에 도달할 수 있게 한다.

## 구성

| 대상 | 위치 | 방식 |
|------|------|------|
| 정책 파일 (tagOwners, autoApprovers) | [`tailscale/`](../../tailscale/) | Terraform (수동 apply) |
| Operator + CRD | [`argocd/applications/infra/tailscale.yaml`](../argocd/applications/infra/tailscale.yaml) | ArgoCD (Helm) |
| Connector (subnet router) | [`manifests/connector.yaml`](manifests/connector.yaml) | ArgoCD |
| OAuth 자격증명 | `manifests/sealed-secret.yaml` | SealedSecret |

Operator 는 `tag:k8s-operator` 로 자기 자신을 등록하고, Connector 가 만드는 subnet
router 파드는 `tag:k8s` 로 등록된다. 광고된 라우트는 정책 파일의 `autoApprovers` 가
자동 승인하므로 콘솔 클릭이 필요 없다.

## 부트스트랩 (최초 1회)

OAuth client 발급만 콘솔에서 해야 한다 — API 자격증명 자체를 API 로 만들 수 없기 때문.
나머지는 전부 코드로 관리된다.

### 1. 태그 선언

Terraform 이 정책 파일을 소유하므로, OAuth client 발급 전에 태그가 존재해야 한다.
[Access controls](https://login.tailscale.com/admin/acls/file) 에서 `tagOwners` 에
`tag:k8s-operator` 와 `tag:k8s` 를 먼저 추가한다. (이후 `terraform apply` 가 이 내용을
포함한 정책 전체를 소유한다.)

### 2. OAuth client 2개 발급

[Trust credentials](https://login.tailscale.com/admin/settings/oauth) 에서 발급한다.
용도를 분리해 최소권한을 유지한다 — operator 용만 클러스터에 상주하므로 노출면이 다르다.

| 용도 | scope | 태그 |
|------|-------|------|
| Terraform (정책 파일 관리) | `all:write` | - |
| Operator (디바이스·키 관리) | `devices:core` write, `auth_keys` write, `services` write | `tag:k8s-operator` |

### 3. Terraform 자격증명을 `.envrc.local` 에 저장

```bash
export TF_VAR_tailscale_oauth_client_id="..."
export TF_VAR_tailscale_oauth_client_secret="..."
```

`.envrc` 의 `TF_VAR_tailscale_tailnet` 도 실제 tailnet 이름으로 채운다
(admin console > Settings > General).

### 4. 정책 파일 apply

```bash
cd tailscale
terraform init
terraform plan
terraform apply
```

기존 정책이 tailnet 기본값이 아니면 `overwrite_existing_content = false` 때문에
apply 가 실패한다. 의도된 안전장치다 — 기존 정책을 [`acl.tf`](../../tailscale/acl.tf)
의 `local.policy` 에 병합한 뒤 `true` 로 바꾼다.

### 5. Operator 자격증명 봉인

VPN/클러스터 접근 없이 봉인 가능하다 (공개키가 레포에 있음).

```bash
kubectl create secret generic operator-oauth \
  --namespace tailscale \
  --from-literal=client_id='<operator OAuth client ID>' \
  --from-literal=client_secret='<operator OAuth client secret>' \
  --dry-run=client -o yaml \
| kubeseal --format yaml --cert k8s/sealed-secrets/cert.pem \
> k8s/tailscale/manifests/sealed-secret.yaml
```

커밋 후 push 하면 ArgoCD 가 sync 한다.

## 클라이언트 설정

라우트는 자동으로 받지 않는다. 각 기기에서 한 번 켜야 한다.

```bash
sudo tailscale up --accept-routes
```

Windows/macOS 는 트레이 메뉴의 "Use Tailscale subnets" 를 켠다.

## 검증

```bash
tailscale status | grep homelab-subnet-router
kubectl --context json get nodes
```

## 되돌리기

```bash
kubectl delete -k k8s/tailscale/manifests   # 또는 ArgoCD Application 삭제
cd tailscale && terraform destroy           # 정책 파일 원복
```

OAuth client 는 콘솔에서 직접 폐기한다.
