terraform {
  backend "s3" {}
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