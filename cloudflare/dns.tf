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

# AMANG project - wildcard subdomains
# Convention: (service).amang.(env).json-server.win
# proxied=false: multi-level subdomains need cert-manager TLS (Cloudflare free Universal SSL doesn't cover)
resource "cloudflare_record" "wildcard_amang" {
  zone_id = cloudflare_zone.main.id
  name    = "*.amang"
  type    = "A"
  content = var.default_ip
  proxied = false
  comment = "AMANG production services (api.amang, s3.amang, minio.amang)"
}

resource "cloudflare_record" "amang_staging" {
  zone_id = cloudflare_zone.main.id
  name    = "amang.staging"
  type    = "CNAME"
  content = "cname.vercel-dns.com"
  proxied = false
  comment = "AMANG staging web (Vercel preview deployment)"
}

resource "cloudflare_record" "wildcard_amang_staging" {
  zone_id = cloudflare_zone.main.id
  name    = "*.amang.staging"
  type    = "A"
  content = var.default_ip
  proxied = false
  comment = "AMANG staging services (api.amang.staging, s3.amang.staging, minio.amang.staging)"
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

resource "cloudflare_record" "vercel_verify_sme_tour" {
  zone_id = cloudflare_zone.main.id
  name    = "_vercel"
  type    = "TXT"
  content = "vc-domain-verify=sme-tour.json-server.win,4959dd229491b907c367"
}

resource "cloudflare_record" "health" {
  zone_id = cloudflare_zone.main.id
  name    = "health"
  type    = "A"
  content = var.default_ip
  proxied = true
  comment = "Health Hub dashboard"
}

# OpenCampus project
resource "cloudflare_record" "opencampus" {
  zone_id = cloudflare_zone.main.id
  name    = "opencampus"
  type    = "A"
  content = var.default_ip
  proxied = true
  comment = "OpenCampus frontend"
}

resource "cloudflare_record" "opencampus_api" {
  zone_id = cloudflare_zone.main.id
  name    = "api.opencampus"
  type    = "A"
  content = var.default_ip
  proxied = false
  comment = "OpenCampus API (proxied=false for 2-level subdomain TLS)"
}

resource "cloudflare_record" "opencampus_s3" {
  zone_id = cloudflare_zone.main.id
  name    = "s3.opencampus"
  type    = "A"
  content = var.default_ip
  proxied = false
  comment = "OpenCampus MinIO S3 (proxied=false for cert-manager)"
}

resource "cloudflare_record" "opencampus_minio" {
  zone_id = cloudflare_zone.main.id
  name    = "minio.opencampus"
  type    = "A"
  content = var.default_ip
  proxied = false
  comment = "OpenCampus MinIO Console (proxied=false for cert-manager)"
}

# SME Tour project
# Convention: (service).sme-tour.json-server.win
resource "cloudflare_record" "sme_tour_api" {
  zone_id = cloudflare_zone.main.id
  name    = "api.sme-tour"
  type    = "A"
  content = var.default_ip
  proxied = false
  comment = "SME Tour engine API (proxied=false for 2-level subdomain TLS)"
}

resource "cloudflare_record" "sme_tour" {
  zone_id = cloudflare_zone.main.id
  name    = "sme-tour"
  type    = "CNAME"
  content = "cname.vercel-dns.com"
  proxied = false
  comment = "SME Tour frontend (Vercel, proxied=false for Vercel SSL)"
}
