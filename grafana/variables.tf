variable "grafana_url" {
  description = "Grafana base URL"
  type        = string
  default     = "https://grafana.json-server.win"
}

variable "grafana_auth" {
  description = "Grafana service account token (Editor 이상)"
  type        = string
  sensitive   = true
}
