terraform {
  required_version = ">= 1.0"

  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "5.27.0"
    }
    http = {
      source  = "hashicorp/http"
      version = "3.5.0"
    }
  }

  # S3 backend - credentials via AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY env vars
  backend "s3" {
    bucket = "homelab-tfstate-361769566809"
    key    = "cloudflare/terraform.tfstate"
    region = "ap-northeast-2"
  }
}

provider "cloudflare" {
  api_token = var.cloudflare_api_token
}
