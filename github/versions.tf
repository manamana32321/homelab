terraform {
  required_version = ">= 1.0"

  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 6.0"
    }
  }

  # S3 backend - credentials via AWS_PROFILE
  backend "s3" {
    bucket = "homelab-tfstate-361769566809"
    key    = "github/terraform.tfstate"
    region = "ap-northeast-2"
  }
}

provider "github" {
  owner = "manamana32321"
  token = var.github_token
}

provider "github" {
  alias = "skku_amang"
  owner = "skku-amang"
  token = var.github_token
}
