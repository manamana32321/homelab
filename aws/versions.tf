terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # S3 backend - credentials via AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY env vars
  backend "s3" {
    bucket = "homelab-tfstate-361769566809"
    key    = "aws/terraform.tfstate"
    region = "ap-northeast-2"
  }
}

provider "aws" {
  region = var.aws_region
}
