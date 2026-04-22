terraform {
  required_version = ">= 1.5"

  required_providers {
    vercel = {
      source  = "vercel/vercel"
      version = "~> 2.0"
    }
  }

  # S3 backend — 기존 homelab-tfstate bucket 재사용
  # credentials: AWS_PROFILE=homelab (.envrc에서 설정)
  backend "s3" {
    bucket = "homelab-tfstate-361769566809"
    key    = "vercel/terraform.tfstate"
    region = "ap-northeast-2"
  }
}

provider "vercel" {
  api_token = var.vercel_api_token
  team      = var.amang_team_id
}
