locals {
  region = get_env("AWS_DEFAULT_REGION")
  stage  = "dev"
}

inputs = {
  stage               = local.stage
  stack               = "dev"
  region              = local.region
  account_id          = "345345345345"
  v1_region           = "us"
  api_domain          = "api.emprovise.com"

  # iam
  deploy_integration_only  = true
  allow_rd_read_ddb        = true
  enable_rd_develop_policy = true

  app_regions = ["us-1"]

  # monitoring
  sns_regions    = ["us-east-1"]

  deploy_artifact_bucket = true

  # vpc cidr
  intra_nat_cidr = "10.10.10.0/24"
  
  slack_channels = {
    "app-bookmarks-api" = "C06UCLHRF7F",
  }
}
