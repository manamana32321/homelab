terraform {
  required_version = ">= 1.0"

  required_providers {
    grafana = {
      source  = "grafana/grafana"
      version = "~> 4.0"
    }
  }

  # S3 backend - credentials via AWS_PROFILE
  backend "s3" {
    bucket = "homelab-tfstate-361769566809"
    key    = "grafana/terraform.tfstate"
    region = "ap-northeast-2"
  }
}

provider "grafana" {
  url  = var.grafana_url
  auth = var.grafana_auth
}
