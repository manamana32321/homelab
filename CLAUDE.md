# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

개인 홈랩 인프라 관리 레포지토리. Cloudflare DNS 및 K3s 클러스터를 선언적으로 관리합니다.

- **도메인**: json-server.win
- **DNS/CDN**: Cloudflare (Terraform으로 관리)
- **클러스터**: K3s + ArgoCD GitOps

## Architecture

```
homelab/
├── cloudflare/              # Terraform - Cloudflare DNS 관리
│   ├── versions.tf          # Provider 및 R2 backend 설정
│   ├── variables.tf         # 변수 정의 (sensitive 포함)
│   ├── dns.tf               # Zone 및 DNS 레코드
│   └── outputs.tf           # Zone ID 등 출력
├── k8s/                     # Kubernetes manifests (ArgoCD GitOps)
│   ├── argocd/
│   │   └── applications/
│   │       ├── infra/           # cert-manager, external-secrets, reflector
│   │       └── observability/   # prometheus, grafana, loki, tempo
│   ├── cloudflare-credentials/  # SealedSecret - Cloudflare API Token
│   ├── cert-manager/            # Let's Encrypt DNS01 (Cloudflare)
│   ├── external-secrets/        # AWS Secrets Manager 연동 (옵션)
│   ├── reflector/               # Secret 복제 설정
│   ├── dashboard/               # Kubernetes Dashboard
│   └── observability/           # 모니터링 스택 (OTel, Prometheus, Loki, Tempo)
├── .envrc                   # 공개 환경변수 (TF_VAR_*)
└── .envrc.local             # 민감한 credentials (gitignored)
```

## Cloudflare/Terraform

### Setup
```bash
# 환경변수 로드
direnv allow

# 초기화 (R2 backend)
cd cloudflare && terraform init

# 변경사항 확인
terraform plan

# 적용
terraform apply
```

### Environment Variables
**.envrc** (committed):
- `TF_VAR_cloudflare_account_id` - Cloudflare 계정 ID
- `TF_VAR_zone_name` - 도메인 (json-server.win)

**.envrc.local** (gitignored):
- `TF_VAR_cloudflare_api_token` - API 토큰 (Zone:DNS:Edit 권한)
- `TF_VAR_default_ip` - 기본 A 레코드 IP
- `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` - R2 접근용
- `AWS_ENDPOINT_URL_S3` - R2 엔드포인트

### Notes
- Cloudflare Email Routing 레코드 (MX route1/2/3, dkim_cf2024)는 Cloudflare가 관리하므로 Terraform에서 제외
- R2는 S3 호환이므로 `backend "s3"` 사용

## Kubernetes (k8s/)

### GitOps with ArgoCD
- 모든 앱은 `k8s/argocd/applications/`에 Application CR로 정의
- Helm 차트는 multi-source 패턴 사용 (chart + values ref)

### Secret Management
- **SealedSecret**: 암호화된 시크릿을 Git에 저장 (`kubeseal`)
- **Reflector**: `cloudflare-credentials` namespace에서 다른 namespace로 복제
- **External Secrets** (옵션): AWS Secrets Manager → K8s Secret 동기화

### Observability Stack
```
App (OTLP) → OTel Collector → Prometheus (메트릭)
                            → Loki (로그)
                            → Tempo (트레이스)
                            → Grafana (대시보드)
```

### 주요 URL
| 서비스 | URL |
|--------|-----|
| ArgoCD | argocd.json-server.win |
| Grafana | grafana.json-server.win |
| Prometheus | prometheus.json-server.win |
| K8s Dashboard | k8s.json-server.win |
| OTel Collector | otel.json-server.win |

## Common Commands

```bash
# Terraform
cd cloudflare && terraform plan
cd cloudflare && terraform apply

# Kubernetes
kubectl get applications -n argocd
kubectl get clustersecretstore
kubectl get externalsecret -A

# SealedSecret 생성
kubeseal --format yaml < secret.yaml > sealed-secret.yaml
```

## Conventions

- 환경변수는 `.envrc` (공개) / `.envrc.local` (민감) 분리
- Terraform 변수는 `TF_VAR_` prefix 사용
- sensitive 변수는 `sensitive = true` 명시
- k8s namespace별 리소스 격리
- DNS01 검증은 Cloudflare API Token 사용
