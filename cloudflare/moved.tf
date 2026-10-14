# provider v4 → v5: cloudflare_record가 cloudflare_dns_record로 이름 변경. state는 v5 provider가 MoveState로 이전
moved {
  from = cloudflare_record.root
  to   = cloudflare_dns_record.root
}

moved {
  from = cloudflare_record.wildcard_amang
  to   = cloudflare_dns_record.wildcard_amang
}

moved {
  from = cloudflare_record.amang_staging
  to   = cloudflare_dns_record.amang_staging
}

moved {
  from = cloudflare_record.wildcard_amang_staging
  to   = cloudflare_dns_record.wildcard_amang_staging
}

moved {
  from = cloudflare_record.argocd
  to   = cloudflare_dns_record.argocd
}

moved {
  from = cloudflare_record.kargo
  to   = cloudflare_dns_record.kargo
}

moved {
  from = cloudflare_record.auth
  to   = cloudflare_dns_record.auth
}

moved {
  from = cloudflare_record.db
  to   = cloudflare_dns_record.db
}

moved {
  from = cloudflare_record.hermes
  to   = cloudflare_dns_record.hermes
}

moved {
  from = cloudflare_record.minecraft
  to   = cloudflare_dns_record.minecraft
}

moved {
  from = cloudflare_record.minecraft_map
  to   = cloudflare_dns_record.minecraft_map
}

moved {
  from = cloudflare_record.k8s
  to   = cloudflare_dns_record.k8s
}

moved {
  from = cloudflare_record.prometheus
  to   = cloudflare_dns_record.prometheus
}

moved {
  from = cloudflare_record.grafana
  to   = cloudflare_dns_record.grafana
}

moved {
  from = cloudflare_record.otel
  to   = cloudflare_dns_record.otel
}

moved {
  from = cloudflare_record.longhorn
  to   = cloudflare_dns_record.longhorn
}

moved {
  from = cloudflare_record.photos
  to   = cloudflare_dns_record.photos
}

moved {
  from = cloudflare_record.files
  to   = cloudflare_dns_record.files
}

moved {
  from = cloudflare_record.amang
  to   = cloudflare_dns_record.amang
}

moved {
  from = cloudflare_record.mx_send
  to   = cloudflare_dns_record.mx_send
}

moved {
  from = cloudflare_record.spf
  to   = cloudflare_dns_record.spf
}

moved {
  from = cloudflare_record.dmarc
  to   = cloudflare_dns_record.dmarc
}

moved {
  from = cloudflare_record.dkim_resend
  to   = cloudflare_dns_record.dkim_resend
}

moved {
  from = cloudflare_record.spf_send
  to   = cloudflare_dns_record.spf_send
}

moved {
  from = cloudflare_record.vercel_verify
  to   = cloudflare_dns_record.vercel_verify
}

moved {
  from = cloudflare_record.health
  to   = cloudflare_dns_record.health
}

moved {
  from = cloudflare_record.essentia
  to   = cloudflare_dns_record.essentia
}

moved {
  from = cloudflare_record.essentia_api
  to   = cloudflare_dns_record.essentia_api
}

moved {
  from = cloudflare_record.essentia_s3
  to   = cloudflare_dns_record.essentia_s3
}

moved {
  from = cloudflare_record.essentia_minio
  to   = cloudflare_dns_record.essentia_minio
}

moved {
  from = cloudflare_record.gbrain
  to   = cloudflare_dns_record.gbrain
}

moved {
  from = cloudflare_record.learningx
  to   = cloudflare_dns_record.learningx
}
