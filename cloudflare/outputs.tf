output "zone_id" {
  description = "Cloudflare Zone ID"
  value       = cloudflare_zone.main.id
}

output "learningx_landing_url" {
  description = "Production URL for the LearningX MCP landing page"
  value       = "https://learningx.${var.zone_name}"
}

output "cert_manager_token" {
  description = "cert-manager DNS01 토큰. SealedSecret 생성용 (tokens.tf 주석 참조)"
  value       = cloudflare_account_token.cert_manager.value
  sensitive   = true
}

output "ddns_token" {
  description = "OpenWrt DDNS 토큰. 공유기 ddns.cf_origin.password 용"
  value       = cloudflare_account_token.ddns.value
  sensitive   = true
}
