output "zone_id" {
  description = "Cloudflare Zone ID"
  value       = cloudflare_zone.main.id
}

output "learningx_landing_url" {
  description = "Production URL for the LearningX MCP landing page"
  value       = "https://learningx.${var.zone_name}"
}
