terraform {
  required_version = ">= 1.0"

  required_providers {
    tailscale = {
      source  = "tailscale/tailscale"
      version = "~> 0.29"
    }
  }

  # S3 backend - credentials via AWS_PROFILE
  backend "s3" {
    bucket = "homelab-tfstate-361769566809"
    key    = "tailscale/terraform.tfstate"
    region = "ap-northeast-2"
  }
}

provider "tailscale" {
  oauth_client_id     = var.tailscale_oauth_client_id
  oauth_client_secret = var.tailscale_oauth_client_secret
  tailnet             = var.tailscale_tailnet
  scopes              = ["all:write"]
}
