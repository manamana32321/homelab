terraform {
  required_version = ">= 1.0"

  required_providers {
    betteruptime = {
      source  = "BetterStackHQ/better-uptime"
      version = "~> 0.13"
    }
  }

  # S3 backend - credentials via AWS_PROFILE
  backend "s3" {
    bucket = "homelab-tfstate-361769566809"
    key    = "betterstack/terraform.tfstate"
    region = "ap-northeast-2"
  }
}

provider "betteruptime" {
  api_token = var.betterstack_api_token
}
