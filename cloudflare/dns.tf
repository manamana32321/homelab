resource "cloudflare_zone" "main" {
  name = var.zone_name
  account = {
    id = var.cloudflare_account_id
  }
}

resource "cloudflare_dns_record" "origin" {
  zone_id = cloudflare_zone.main.id
  name    = "origin"
  type    = "A"
  content = var.default_ip
  proxied = false
  comment = "집 공인 IP — OpenWrt DDNS 가 갱신"
  ttl     = 60

  lifecycle {
    ignore_changes = [content]
  }
}

resource "cloudflare_dns_record" "root" {
  zone_id = cloudflare_zone.main.id
  name    = "json-server.win"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = true
  comment = "IpTime A1004"
  ttl     = 1
}

# AMANG project - wildcard subdomains
# Convention: (service).amang.(env).json-server.win
# proxied=false: multi-level subdomains need cert-manager TLS (Cloudflare free Universal SSL doesn't cover)
resource "cloudflare_dns_record" "wildcard_amang" {
  zone_id = cloudflare_zone.main.id
  name    = "*.amang"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = false
  comment = "AMANG production services (api.amang, s3.amang, minio.amang)"
  ttl     = 1
}

resource "cloudflare_dns_record" "amang_staging" {
  zone_id = cloudflare_zone.main.id
  name    = "amang.staging"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = false
  comment = "AMANG staging web (홈랩 K3s, skku-amang/main#749). 롤백: CNAME cname.vercel-dns.com"
  ttl     = 1
}

resource "cloudflare_dns_record" "wildcard_amang_staging" {
  zone_id = cloudflare_zone.main.id
  name    = "*.amang.staging"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = false
  comment = "AMANG staging services (api.amang.staging, s3.amang.staging, minio.amang.staging)"
  ttl     = 1
}

resource "cloudflare_dns_record" "argocd" {
  zone_id = cloudflare_zone.main.id
  name    = "argocd"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = true
  ttl     = 1
}

resource "cloudflare_dns_record" "kargo" {
  zone_id = cloudflare_zone.main.id
  name    = "kargo"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = true
  comment = "Kargo 환경 승격 UI"
  ttl     = 1
}

resource "cloudflare_dns_record" "auth" {
  zone_id = cloudflare_zone.main.id
  name    = "auth"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = true
  comment = "Authentik SSO"
  ttl     = 1
}

resource "cloudflare_dns_record" "db" {
  zone_id = cloudflare_zone.main.id
  name    = "db"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = true
  comment = "CloudBeaver DB admin"
  ttl     = 1
}

resource "cloudflare_dns_record" "hermes" {
  zone_id = cloudflare_zone.main.id
  name    = "hermes"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = true
  comment = "Hermes Agent admin dashboard (Authentik forward-auth)"
  ttl     = 1
}

resource "cloudflare_dns_record" "minecraft" {
  zone_id = cloudflare_zone.main.id
  name    = "mc"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = false
  ttl     = 1
}

resource "cloudflare_dns_record" "minecraft_map" {
  zone_id = cloudflare_zone.main.id
  name    = "mcmap"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = true
  comment = "BlueMap 3D web viewer for Minecraft"
  ttl     = 1
}

resource "cloudflare_dns_record" "k8s" {
  zone_id = cloudflare_zone.main.id
  name    = "k8s"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = true
  comment = "Headlamp"
  ttl     = 1
}

resource "cloudflare_dns_record" "prometheus" {
  zone_id = cloudflare_zone.main.id
  name    = "prometheus"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = true
  ttl     = 1
}

resource "cloudflare_dns_record" "grafana" {
  zone_id = cloudflare_zone.main.id
  name    = "grafana"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = true
  ttl     = 1
}

resource "cloudflare_dns_record" "otel" {
  zone_id = cloudflare_zone.main.id
  name    = "otel"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = true
  ttl     = 1
}


resource "cloudflare_dns_record" "longhorn" {
  zone_id = cloudflare_zone.main.id
  name    = "longhorn"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = true
  comment = "Longhorn storage UI"
  ttl     = 1
}

resource "cloudflare_dns_record" "photos" {
  zone_id = cloudflare_zone.main.id
  name    = "photos"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = false
  comment = "Immich photo management"
  ttl     = 1
}

resource "cloudflare_dns_record" "files" {
  zone_id = cloudflare_zone.main.id
  name    = "files"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = false
  comment = "Seafile file sync"
  ttl     = 1
}

# CNAME Records
resource "cloudflare_dns_record" "amang" {
  zone_id = cloudflare_zone.main.id
  name    = "amang"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = false
  comment = "AMANG production web (홈랩 K3s, skku-amang/main#749). 롤백: CNAME 0e652236f019368a.vercel-dns-017.com"
  ttl     = 600
}

# MX Records
# Note: mx_route1, mx_route2, mx_route3 are managed by Cloudflare Email Routing

resource "cloudflare_dns_record" "mx_send" {
  zone_id  = cloudflare_zone.main.id
  name     = "send"
  type     = "MX"
  content  = "feedback-smtp.ap-northeast-1.amazonses.com"
  priority = 10
  ttl      = 1
}

# TXT Records
resource "cloudflare_dns_record" "spf" {
  zone_id = cloudflare_zone.main.id
  name    = "json-server.win"
  type    = "TXT"
  content = "v=spf1 include:_spf.mx.cloudflare.net ~all"
  ttl     = 1
}

resource "cloudflare_dns_record" "dmarc" {
  zone_id = cloudflare_zone.main.id
  name    = "_dmarc"
  type    = "TXT"
  content = "v=DMARC1; p=none;"
  ttl     = 1
}

# Note: dkim_cf2024 is managed by Cloudflare Email Routing

resource "cloudflare_dns_record" "dkim_resend" {
  zone_id = cloudflare_zone.main.id
  name    = "resend._domainkey"
  type    = "TXT"
  content = "p=MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDGWktCLSn1THZNgpyXIzNGI+bjGeOCuRRaPWQlJ186+qlChCl7VQEKhf4pgamM6z+tWZPHM8VtimcBfXSG47tTB/+EWWmKjwXUI+QA7KiL0kpneWSspUrPRcnX0WgnVFYAEh6zE6sN7dgENk2nx+lHvAzxA5tPUkrUyBmb8yoijwIDAQAB"
  ttl     = 1
}

resource "cloudflare_dns_record" "spf_send" {
  zone_id = cloudflare_zone.main.id
  name    = "send"
  type    = "TXT"
  content = "v=spf1 include:amazonses.com ~all"
  ttl     = 1
}

resource "cloudflare_dns_record" "vercel_verify" {
  zone_id = cloudflare_zone.main.id
  name    = "_vercel"
  type    = "TXT"
  content = "vc-domain-verify=amang.json-server.win,9fa5cbcbc9713e7db0c9,dc"
  ttl     = 600
}

resource "cloudflare_dns_record" "health" {
  zone_id = cloudflare_zone.main.id
  name    = "health"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = true
  comment = "Health Hub dashboard"
  ttl     = 1
}

# Essentia project (rebranded from Saemate, 2026-04-19)
resource "cloudflare_dns_record" "essentia" {
  zone_id = cloudflare_zone.main.id
  name    = "essentia"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = true
  comment = "Essentia frontend"
  ttl     = 1
}

resource "cloudflare_dns_record" "essentia_api" {
  zone_id = cloudflare_zone.main.id
  name    = "api.essentia"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = false
  comment = "Essentia API (proxied=false for 2-level subdomain TLS)"
  ttl     = 1
}

resource "cloudflare_dns_record" "essentia_s3" {
  zone_id = cloudflare_zone.main.id
  name    = "s3.essentia"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = false
  comment = "Essentia MinIO S3 (proxied=false for cert-manager)"
  ttl     = 1
}

resource "cloudflare_dns_record" "essentia_minio" {
  zone_id = cloudflare_zone.main.id
  name    = "minio.essentia"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = false
  comment = "Essentia MinIO Console (proxied=false for cert-manager)"
  ttl     = 1
}

# gbrain MCP (HTTP MCP server for Claude custom connector)
resource "cloudflare_dns_record" "gbrain" {
  zone_id = cloudflare_zone.main.id
  name    = "gbrain"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = true
  comment = "gbrain HTTP MCP — Bearer auth (caddy sidecar), Cloudflare proxied"
  ttl     = 1
}

# LearningX MCP 마케팅 landing. Cloudflare Pages 정적 사이트에서 essentia-web
# ingress로 이전 (essentia-edu/essentia#83). apps/web의 (marketing)/learningx
# route를 host-based rewrite로 서빙 — essentia-web ingress가 이 host를 받는다.
resource "cloudflare_dns_record" "learningx" {
  zone_id = cloudflare_zone.main.id
  name    = "learningx"
  type    = "CNAME"
  content = "origin.${var.zone_name}"
  proxied = true
  comment = "LearningX MCP landing — essentia-web ingress (Next host rewrite)"
  ttl     = 1
}
