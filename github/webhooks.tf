locals {
  argocd_webhook_url = "https://argocd.json-server.win/api/webhook"
}

resource "github_repository_webhook" "argocd_homelab" {
  repository = "homelab"
  events     = ["push"]

  configuration {
    url          = local.argocd_webhook_url
    content_type = "json"
    secret       = var.argocd_webhook_secret
    insecure_ssl = false
  }
}

resource "github_repository_webhook" "argocd_amang" {
  provider   = github.skku_amang
  repository = "main"
  events     = ["push"]

  configuration {
    url          = local.argocd_webhook_url
    content_type = "json"
    secret       = var.argocd_webhook_secret
    insecure_ssl = false
  }
}
