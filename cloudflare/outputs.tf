output "zone_id" {
  description = "Cloudflare Zone ID"
  value       = cloudflare_zone.main.id
}

output "learningx_landing_url" {
  description = "Production URL for the LearningX MCP landing page"
  value       = "https://learningx.${var.zone_name}"
}

output "cert_manager_token" {
  description = "cert-manager DNS01 토큰. SealedSecret 생성용 (k8s/cloudflare-credentials/README.md)"
  value       = cloudflare_account_token.cert_manager.value
  sensitive   = true
}
