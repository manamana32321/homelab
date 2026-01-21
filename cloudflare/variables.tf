variable "cloudflare_api_token" {
  description = "Cloudflare API Token"
  type        = string
  sensitive   = true
}

variable "cloudflare_account_id" {
  description = "Cloudflare Account ID"
  type        = string
}

variable "zone_name" {
  description = "Domain name to manage"
  type        = string
}

variable "default_ip" {
  description = "Default IP address for A records"
  type        = string
  sensitive   = true
}
