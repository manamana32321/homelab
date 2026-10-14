# cert-manager DNS01 검증용 토큰 (letsencrypt-dns01 ClusterIssuer)
# 값은 k8s/cloudflare-credentials/cloudflare-api-token.yaml SealedSecret으로 들어감. 재발급 절차는 그 디렉토리 README 참조
data "cloudflare_account_api_token_permission_groups_list" "zone_read" {
  account_id = var.cloudflare_account_id
  name       = "Zone Read"
}

data "cloudflare_account_api_token_permission_groups_list" "dns_write" {
  account_id = var.cloudflare_account_id
  name       = "DNS Write"
}

resource "cloudflare_account_token" "cert_manager" {
  account_id = var.cloudflare_account_id
  name       = "cert-manager DNS01 (homelab)"

  policies = [{
    effect = "allow"
    permission_groups = [
      { id = data.cloudflare_account_api_token_permission_groups_list.zone_read.result[0].id },
      { id = data.cloudflare_account_api_token_permission_groups_list.dns_write.result[0].id },
    ]
    resources = jsonencode({
      "com.cloudflare.api.account.zone.${cloudflare_zone.main.id}" = "*"
    })
  }]
}
