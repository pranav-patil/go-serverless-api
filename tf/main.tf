terraform {
  backend "s3" {}
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "4.27.0"
    }
  }
}

locals {
  tags = {
    Stack     = var.stack
    Stage     = var.stage
    CreatedBy = var.user
  }
}

provider "aws" {
  alias  = "us"
  region = "us-east-1"
}
