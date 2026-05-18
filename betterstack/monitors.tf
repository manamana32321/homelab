###############################################################################
# External uptime probe for argocd.json-server.win
#
# Replaces the Coral Dev Board external blackbox (retired 2026-05-18, PR #219).
# Argo CD is the canonical external endpoint because it shares DNS, Cloudflare,
# ingress controller, and TLS cert issuance with every other public service —
# a fault in those layers will surface here first.
#
# Free plan: 10 monitors / 3-min check interval / multi-region default.
# Notifications routed to Telegram via the Telegram integration registered
# in the BetterStack UI (default escalation policy fan-outs to every active
# integration; no policy_id needed).
###############################################################################

resource "betteruptime_monitor" "argocd" {
  url                = "https://argocd.json-server.win"
  monitor_type       = "status"
  pronounceable_name = "Argo CD (json-server.win)"

  # Free-plan minimum interval. 3 fails over ~9 min before incident opens.
  check_frequency     = 180
  request_timeout     = 30
  recovery_period     = 0
  confirmation_period = 60

  # 2xx and 3xx are both fine — Argo CD redirects unauthenticated GET / to /login.
  http_method           = "GET"
  expected_status_codes = [200, 301, 302]
  follow_redirects      = true
  remember_cookies      = false

  # TLS certificate watchdog (cert-manager renews at 30d remaining; alert at 7d as
  # a fail-safe in case renewal stalls).
  verify_ssl     = true
  ssl_expiration = 7

  # Domain registry expiry is managed by Cloudflare auto-renew; 30d is a buffer
  # in case auto-renew silently fails.
  domain_expiration = 30
}

output "argocd_monitor_id" {
  description = "Better Stack monitor ID for argocd.json-server.win"
  value       = betteruptime_monitor.argocd.id
}

output "argocd_monitor_url" {
  description = "Direct URL to the monitor in Better Stack UI"
  value       = "https://uptime.betterstack.com/team/monitors/${betteruptime_monitor.argocd.id}"
}
