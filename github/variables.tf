variable "github_token" {
  description = "GitHub token with repo scope (admin on target repos)"
  type        = string
  sensitive   = true
}

variable "argocd_webhook_secret" {
  description = "HMAC secret shared with SealedSecret argocd/argocd-webhook-github"
  type        = string
  sensitive   = true
}
