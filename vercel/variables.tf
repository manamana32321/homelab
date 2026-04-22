variable "vercel_api_token" {
  description = "Vercel API token (https://vercel.com/account/tokens). Set in .envrc.local via TF_VAR_vercel_api_token."
  type        = string
  sensitive   = true
}

variable "amang_team_id" {
  description = "Amang team slug/id. sme-tour는 Vercel Hobby 정책상 personal 이동 불가라 한시적으로 Amang Pro scope에 기거 중."
  type        = string
  default     = "team_aDiKenl6xun665nVWV49POd4"
}

variable "sme_tour_api_base" {
  description = "Public API base URL injected into sme-tour Next.js frontend as NEXT_PUBLIC_API_BASE. K8s ingress에 배포된 sme-tour-engine."
  type        = string
  default     = "https://api.sme-tour.json-server.win"
}
