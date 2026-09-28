import {
  to = cloudflare_ruleset.waf_custom
  id = "zone/${cloudflare_zone.main.id}/0754ef0ae97b4974b22d5c6b38beb82c"
}

resource "cloudflare_ruleset" "waf_custom" {
  zone_id = cloudflare_zone.main.id
  name    = "default"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  rules {
    ref         = "github_webhook_argocd"
    description = "GitHub webhook → ArgoCD 허용"
    expression  = "(http.host eq \"argocd.json-server.win\" and http.request.uri.path eq \"/api/webhook\" and http.request.method eq \"POST\" and ip.src.asnum eq 36459)"
    action      = "skip"
    action_parameters {
      ruleset = "current"
    }
    logging {
      enabled = true
    }
    enabled = true
  }

  rules {
    ref         = "8609e939524444ca86b3eb8ea6dac776"
    description = "Amang API 차단 허용"
    expression  = "(http.host wildcard r\"*.json-server.win\")"
    action      = "skip"
    action_parameters {
      phases = ["http_ratelimit"]
    }
    logging {
      enabled = true
    }
    enabled = false
  }

  rules {
    ref         = "31f6cb807572460dbe3edc7046d40612"
    description = "대한민국만 허용"
    expression  = "(ip.src.country ne \"KR\")"
    action      = "block"
    enabled     = true
  }
}
