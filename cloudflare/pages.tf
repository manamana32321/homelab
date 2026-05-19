# Cloudflare Pages — @essentia-edu/learningx-mcp 마케팅 landing.
#
# 정적 export (Next.js `output: "export"`)된 산출물을 Cloudflare 엣지에서 서빙.
# 서버 인프라 0대, build/bandwidth 무료 한도(500 builds/월, unlimited bandwidth)
# 내에서 처리. PR마다 preview deploy 자동 발급.
#
# Prerequisite (한 번만 콘솔에서):
# Cloudflare 대시보드 → Workers & Pages → Connect to GitHub → essentia-edu org
# install. cloudflare provider는 OAuth 자체를 만들지 못하므로 첫 연결만 수동.
# 한 번 연결되면 이후 project 추가는 본 terraform이 선언적으로 관리.

resource "cloudflare_pages_project" "learningx_landing" {
  account_id        = var.cloudflare_account_id
  name              = "essentia-learningx"
  production_branch = "main"

  source {
    type = "github"
    config {
      owner                         = "essentia-edu"
      repo_name                     = "essentia"
      production_branch             = "main"
      pr_comments_enabled           = true
      deployments_enabled           = true
      production_deployment_enabled = true
      preview_deployment_setting    = "all"
      preview_branch_includes       = ["*"]
      preview_branch_excludes       = []
    }
  }

  build_config {
    # 모노레포 root에서 install + landing app만 build. 빌드 시간 단축을 위해
    # apps/api, apps/web은 빌드하지 않음.
    build_command   = "pnpm install --frozen-lockfile && pnpm --filter @essentia/landing build"
    destination_dir = "apps/landing/out"
    root_dir        = ""
  }

  deployment_configs {
    production {
      environment_variables = {
        NODE_VERSION = "22"
      }
      compatibility_date  = "2025-11-01"
      compatibility_flags = ["nodejs_compat"]
    }
    preview {
      environment_variables = {
        NODE_VERSION = "22"
      }
      compatibility_date  = "2025-11-01"
      compatibility_flags = ["nodejs_compat"]
    }
  }
}

resource "cloudflare_pages_domain" "learningx_landing" {
  account_id   = var.cloudflare_account_id
  project_name = cloudflare_pages_project.learningx_landing.name
  domain       = "learningx.${var.zone_name}"
}

# DNS CNAME → Cloudflare Pages project subdomain.
# proxied = true: Cloudflare가 SSL/CDN 자동 처리 (Universal SSL이 single-level
# subdomain은 커버). multi-level (예: a.b.json-server.win)이었으면
# proxied=false + cert-manager DNS01 였을 것.
resource "cloudflare_record" "learningx_landing" {
  zone_id = cloudflare_zone.main.id
  name    = "learningx"
  type    = "CNAME"
  content = cloudflare_pages_project.learningx_landing.subdomain
  proxied = true
  comment = "@essentia-edu/learningx landing (Cloudflare Pages)"
}
