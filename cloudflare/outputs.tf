output "zone_id" {
  description = "Cloudflare Zone ID"
  value       = cloudflare_zone.main.id
}

output "learningx_landing_subdomain" {
  description = "Cloudflare Pages default subdomain (e.g. essentia-learningx.pages.dev)"
  value       = cloudflare_pages_project.learningx_landing.subdomain
}

output "learningx_landing_url" {
  description = "Production URL for the LearningX MCP landing page"
  value       = "https://learningx.${var.zone_name}"
}
