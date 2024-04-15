

module "kms_dynamo" {
  source = "./modules/kms"

  name       = "KMS-DYNAMO"
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

module "kms_lambda_env" {
  source = "./modules/kms"

  name       = "KMS-LAMBDA-ENV"
  account_id = var.account_id

  description              = "KMS key for lambda environment variable encryption"
  customer_master_key_spec = "SYMMETRIC_DEFAULT"
  key_usage                = "ENCRYPT_DECRYPT"
  enable_key_rotation      = true
  is_enabled               = true
  tags                     = local.tags
  multi_region             = false
}
