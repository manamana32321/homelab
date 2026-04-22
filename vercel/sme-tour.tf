# SME-Tour Vercel 프로젝트 — 종합설계 팀 프로젝트
#
# scope: Amang team (Hobby 제약으로 personal 이동 불가, PR #143 이관)
# lifecycle: 학기 종료 후 `terraform destroy` 로 제거 예정
#
# Import 기존 프로젝트 (최초 `terraform apply` 전 1회만):
#   terraform import \
#     -var="vercel_api_token=$TF_VAR_vercel_api_token" \
#     vercel_project.sme_tour \
#     prj_AM3ftVU9rEv1rhdfpENy4T8l9bDq
#
#   terraform import \
#     -var="vercel_api_token=$TF_VAR_vercel_api_token" \
#     vercel_project_environment_variable.sme_tour_api_base \
#     prj_AM3ftVU9rEv1rhdfpENy4T8l9bDq/14squVanlmQYCB2V

resource "vercel_project" "sme_tour" {
  name      = "sme-tour"
  framework = "nextjs"

  git_repository = {
    type              = "github"
    repo              = "manamana32321/sme-tour"
    production_branch = "main"
  }

  # monorepo-ish 구조: docs/engine/frontend/k8s. Next.js는 frontend/ 에만 존재.
  root_directory = "frontend"
  node_version   = "24.x"

  # Framework default 따름 (next build / .next / next dev).
  # build_command / install_command / dev_command / output_directory 모두 null.
}

resource "vercel_project_environment_variable" "sme_tour_api_base" {
  project_id = vercel_project.sme_tour.id
  key        = "NEXT_PUBLIC_API_BASE"
  value      = var.sme_tour_api_base
  target     = ["production", "preview"]
  comment    = "K8s ingress에 배포된 sme-tour-engine API base URL"
}
