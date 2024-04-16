
module "kms_dynamo" {
  source = "./modules/kms"

  name       = "KMS-DYNAMODB"
  account_id = var.account_id

  description              = "KMS key for dynamo encryption"
  customer_master_key_spec = "SYMMETRIC_DEFAULT"
  key_usage                = "ENCRYPT_DECRYPT"
  enable_key_rotation      = true
  is_enabled               = true
  tags                     = local.tags
  multi_region             = false
}

module "kms_s3" {
  source = "./modules/kms"

  name       = "KMS-S3"
  account_id = var.account_id

  description              = "KMS key for S3 encryption"
  customer_master_key_spec = "SYMMETRIC_DEFAULT"
  key_usage                = "ENCRYPT_DECRYPT"
  enable_key_rotation      = true
  is_enabled               = true
  tags                     = local.tags
  multi_region             = false
}

module "kms_sqs" {
  source = "./modules/kms"

  name       = "KMS-SQS"
  account_id = var.account_id

  description              = "KMS key for SQS encryption"
  customer_master_key_spec = "SYMMETRIC_DEFAULT"
  key_usage                = "ENCRYPT_DECRYPT"
  enable_key_rotation      = true
  is_enabled               = true
  tags                     = local.tags
  multi_region             = false
}

module "kms_sns" {
  source = "./modules/kms"

  name       = "KMS-SNS"
  account_id = var.account_id

  description              = "KMS key for SNS encryption"
  customer_master_key_spec = "SYMMETRIC_DEFAULT"
  key_usage                = "ENCRYPT_DECRYPT"
  enable_key_rotation      = true
  is_enabled               = true
  tags                     = local.tags
  multi_region             = false
}

module "kms_lambda" {
  source = "./modules/kms"

  name       = "KMS-lambda"
  account_id = var.account_id

  description              = "KMS key for Lambda encryption"
  customer_master_key_spec = "SYMMETRIC_DEFAULT"
  key_usage                = "ENCRYPT_DECRYPT"
  enable_key_rotation      = true
  is_enabled               = true
  tags                     = local.tags
  multi_region             = false
}

module "kms_serverless_bucket" {
  source = "./modules/kms"

  name       = "serverless-bucket"
  account_id = var.account_id

  description              = "KMS key for lambda serverless bucket"
  customer_master_key_spec = "SYMMETRIC_DEFAULT"
  key_usage                = "ENCRYPT_DECRYPT"
  enable_key_rotation      = true
  is_enabled               = true
  tags                     = local.tags
  multi_region             = false
}
