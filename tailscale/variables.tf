variable "tailscale_tailnet" {
  description = "Tailscale tailnet. \"-\" 는 OAuth client 가 속한 기본 tailnet 으로 해석된다."
  type        = string
  default     = "-"
}

variable "tailscale_oauth_client_id" {
  description = "Terraform 전용 OAuth client ID (scope: all:write)"
  type        = string
  sensitive   = true
}

variable "tailscale_oauth_client_secret" {
  description = "Terraform 전용 OAuth client secret"
  type        = string
  sensitive   = true
}

variable "homelab_subnet" {
  description = "subnet router 가 tailnet 에 광고할 홈랩 LAN 대역"
  type        = string
  default     = "192.168.0.0/24"
}
