resource "cloudflare_zone" "main" {
  account_id = var.cloudflare_account_id
  zone       = var.zone_name
}

resource "cloudflare_record" "root" {
  zone_id = cloudflare_zone.main.id
  name    = "json-server.win"
  type    = "A"
  content = var.default_ip
  proxied = true
  comment = "IpTime A1004"
}

resource "cloudflare_record" "amang_api" {
  zone_id = cloudflare_zone.main.id
  name    = "amang-api"
  type    = "A"
  content = var.default_ip
  proxied = true
}

resource "cloudflare_record" "amang_api_staging" {
  zone_id = cloudflare_zone.main.id
  name    = "amang-api-staging"
  type    = "A"
  content = var.default_ip
  proxied = true
}

resource "cloudflare_record" "amang_staging" {
  zone_id = cloudflare_zone.main.id
  name    = "amang-staging"
  type    = "A"
  content = var.default_ip
  proxied = true
}

resource "cloudflare_record" "argocd" {
  zone_id = cloudflare_zone.main.id
  name    = "argocd"
  type    = "A"
  content = var.default_ip
  proxied = true
}

resource "cloudflare_record" "auth" {
  zone_id = cloudflare_zone.main.id
  name    = "auth"
  type    = "A"
  content = var.default_ip
  proxied = true
  comment = "Authentik SSO"
}

resource "cloudflare_record" "db" {
  zone_id = cloudflare_zone.main.id
  name    = "db"
  type    = "A"
  content = var.default_ip
  proxied = true
  comment = "CloudBeaver DB admin"
}

resource "cloudflare_record" "factorio_admin" {
  zone_id = cloudflare_zone.main.id
  name    = "factorio-admin"
  type    = "A"
  content = var.default_ip
  proxied = true
}

resource "cloudflare_record" "factorio" {
  zone_id = cloudflare_zone.main.id
  name    = "factorio"
  type    = "A"
  content = var.default_ip
  proxied = false
}

resource "cloudflare_record" "factorio_rcon" {
  zone_id = cloudflare_zone.main.id
  name    = "factorio-rcon"
  type    = "A"
  content = var.default_ip
  proxied = false
  comment = "Factorio RCON (TCP 30100)"
}

resource "cloudflare_record" "factorio_minio_console" {
  zone_id = cloudflare_zone.main.id
  name    = "factorio-minio"
  type    = "A"
  content = var.default_ip
  proxied = true
  comment = "Factorio MinIO Console"
}

resource "cloudflare_record" "minecraft" {
  zone_id = cloudflare_zone.main.id
  name    = "mc"
  type    = "A"
  content = var.default_ip
  proxied = false
}

resource "cloudflare_record" "k8s" {
  zone_id = cloudflare_zone.main.id
  name    = "k8s"
  type    = "A"
  content = var.default_ip
  proxied = false
  comment = "Headlamp - proxy off to avoid Cloudflare challenge on API calls"
}

resource "cloudflare_record" "prometheus" {
  zone_id = cloudflare_zone.main.id
  name    = "prometheus"
  type    = "A"
  content = var.default_ip
  proxied = true
}

resource "cloudflare_record" "grafana" {
  zone_id = cloudflare_zone.main.id
  name    = "grafana"
  type    = "A"
  content = var.default_ip
  proxied = true
}

resource "cloudflare_record" "otel" {
  zone_id = cloudflare_zone.main.id
  name    = "otel"
  type    = "A"
  content = var.default_ip
  proxied = true
}

resource "cloudflare_record" "s3" {
  zone_id = cloudflare_zone.main.id
  name    = "s3"
  type    = "A"
  content = var.default_ip
  proxied = true
  comment = "MinIO S3 API (production)"
}

resource "cloudflare_record" "s3_staging" {
  zone_id = cloudflare_zone.main.id
  name    = "s3-staging"
  type    = "A"
  content = var.default_ip
  proxied = true
  comment = "MinIO S3 API (staging)"
}

resource "cloudflare_record" "amang_minio_console" {
  zone_id = cloudflare_zone.main.id
  name    = "amang-minio-console"
  type    = "A"
  content = var.default_ip
  proxied = true
  comment = "AMANG MinIO Console (production)"
}

resource "cloudflare_record" "amang_minio_console_staging" {
  zone_id = cloudflare_zone.main.id
  name    = "amang-minio-console-staging"
  type    = "A"
  content = var.default_ip
  proxied = true
  comment = "AMANG MinIO Console (staging)"
}

resource "cloudflare_record" "claw" {
  zone_id = cloudflare_zone.main.id
  name    = "claw"
  type    = "A"
  content = var.default_ip
  proxied = true
  comment = "OpenClaw AI Agent"
}

resource "cloudflare_record" "longhorn" {
  zone_id = cloudflare_zone.main.id
  name    = "longhorn"
  type    = "A"
  content = var.default_ip
  proxied = false
  comment = "Longhorn storage UI"
}

resource "cloudflare_record" "frigate" {
  zone_id = cloudflare_zone.main.id
  name    = "frigate"
  type    = "A"
  content = var.default_ip
  proxied = false
  comment = "Frigate NVR"
}

resource "cloudflare_record" "home_assistant" {
  zone_id = cloudflare_zone.main.id
  name    = "ha"
  type    = "A"
  content = var.default_ip
  proxied = false
  comment = "Home Assistant"
}

resource "cloudflare_record" "photos" {
  zone_id = cloudflare_zone.main.id
  name    = "photos"
  type    = "A"
  content = var.default_ip
  proxied = false
  comment = "Immich photo management"
}

resource "cloudflare_record" "files" {
  zone_id = cloudflare_zone.main.id
  name    = "files"
  type    = "A"
  content = var.default_ip
  proxied = false
  comment = "Seafile file sync"
}

# CNAME Records
resource "cloudflare_record" "amang" {
  zone_id = cloudflare_zone.main.id
  name    = "amang"
  type    = "CNAME"
  content = "0e652236f019368a.vercel-dns-017.com"
  proxied = false
}

# MX Records
# Note: mx_route1, mx_route2, mx_route3 are managed by Cloudflare Email Routing

resource "cloudflare_record" "mx_send" {
  zone_id  = cloudflare_zone.main.id
  name     = "send"
  type     = "MX"
  content  = "feedback-smtp.ap-northeast-1.amazonses.com"
  priority = 10
}

# TXT Records
resource "cloudflare_record" "spf" {
  zone_id = cloudflare_zone.main.id
  name    = "json-server.win"
  type    = "TXT"
  content = "v=spf1 include:_spf.mx.cloudflare.net ~all"
}

resource "cloudflare_record" "dmarc" {
  zone_id = cloudflare_zone.main.id
  name    = "_dmarc"
  type    = "TXT"
  content = "v=DMARC1; p=none;"
}

# Note: dkim_cf2024 is managed by Cloudflare Email Routing

resource "cloudflare_record" "dkim_resend" {
  zone_id = cloudflare_zone.main.id
  name    = "resend._domainkey"
  type    = "TXT"
  content = "p=MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDGWktCLSn1THZNgpyXIzNGI+bjGeOCuRRaPWQlJ186+qlChCl7VQEKhf4pgamM6z+tWZPHM8VtimcBfXSG47tTB/+EWWmKjwXUI+QA7KiL0kpneWSspUrPRcnX0WgnVFYAEh6zE6sN7dgENk2nx+lHvAzxA5tPUkrUyBmb8yoijwIDAQAB"
}

resource "cloudflare_record" "spf_send" {
  zone_id = cloudflare_zone.main.id
  name    = "send"
  type    = "TXT"
  content = "v=spf1 include:amazonses.com ~all"
}

resource "cloudflare_record" "vercel_verify" {
  zone_id = cloudflare_zone.main.id
  name    = "_vercel"
  type    = "TXT"
  content = "vc-domain-verify=amang.json-server.win,9fa5cbcbc9713e7db0c9,dc"
}
