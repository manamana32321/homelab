import {
  to = cloudflare_ruleset.waf_custom
  id = "zone/${cloudflare_zone.main.id}/0754ef0ae97b4974b22d5c6b38beb82c"
}

data "http" "github_meta" {
  url = "https://api.github.com/meta"

  request_headers = {
    Accept = "application/vnd.github+json"
  }

  lifecycle {
    postcondition {
      condition     = self.status_code == 200 && length(jsondecode(self.response_body).hooks) > 0
      error_message = "GitHub meta API 응답에 hooks 대역이 없음"
    }
  }
}

locals {
  github_hooks_cidrs = jsondecode(data.http.github_meta.response_body).hooks
}

resource "cloudflare_ruleset" "waf_custom" {
  zone_id = cloudflare_zone.main.id
  name    = "default"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  rules = [
    {
      ref         = "github_webhook_argocd"
      description = "GitHub webhook → ArgoCD 허용"
      expression  = "(http.host eq \"argocd.json-server.win\" and http.request.uri.path eq \"/api/webhook\" and http.request.method eq \"POST\" and ip.src in {${join(" ", local.github_hooks_cidrs)}})"
      action      = "skip"
      enabled     = true
      action_parameters = {
        ruleset = "current"
      }
      logging = {
        enabled = true
      }
    },
    {
      ref         = "31f6cb807572460dbe3edc7046d40612"
      description = "대한민국만 허용"
      expression  = "(ip.src.country ne \"KR\")"
      action      = "block"
      enabled     = true
    }
  ]
}
