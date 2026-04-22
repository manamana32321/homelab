# Vercel — Terraform IaC

homelab에서 관리하는 Vercel 리소스. `cloudflare/`, `aws/` 패턴 동일.

## 관리 대상

| 리소스 | scope | 비고 |
|---|---|---|
| `vercel_project.sme_tour` | Amang team (`test-576d510a`) | 종합설계 한시적, 학기 후 destroy |

## 사전 요구

1. Vercel API token 발급
   - https://vercel.com/account/tokens → Create Token
   - Scope: **Full Account** (Amang team 포함 전체)
   - Expiration: 90일 권장 (과제 기간 충분)
2. `.envrc.local` 에 추가:
   ```bash
   export TF_VAR_vercel_api_token="<token>"
   ```
3. `direnv allow`

## 최초 셋업 (import 1회)

```bash
cd vercel
terraform init

# 기존 Vercel 리소스를 Terraform state로 편입
terraform import vercel_project.sme_tour prj_AM3ftVU9rEv1rhdfpENy4T8l9bDq
terraform import \
  vercel_project_environment_variable.sme_tour_api_base \
  prj_AM3ftVU9rEv1rhdfpENy4T8l9bDq/14squVanlmQYCB2V

# 변경 없어야 정상. 변경 뜨면 코드 조정 후 커밋
terraform plan
```

Import 후엔 `terraform plan/apply` 정상 사용.

## 일상 운영

```bash
terraform plan   # drift 체크
terraform apply  # 변경 적용 (CLAUDE.md 규칙: -auto-approve 금지)
```

## 변경 예시

### env 값 변경

`variables.tf` 의 `sme_tour_api_base` default 수정 또는 tfvars 사용. `terraform apply`.

### custom domain 추가

`sme-tour.tf` 에 아래 block 추가:

```hcl
resource "vercel_project_domain" "sme_tour_custom" {
  project_id = vercel_project.sme_tour.id
  domain     = "tour.example.com"
}
```

### 학기 종료 후 제거

```bash
terraform destroy
```

Vercel UI에서 수동 삭제하지 말 것 — state 불일치 방지.

## State

- Backend: S3 (`homelab-tfstate-361769566809/vercel/terraform.tfstate`, ap-northeast-2)
- Credentials: `AWS_PROFILE=homelab` (상위 `.envrc`에서 export됨)
