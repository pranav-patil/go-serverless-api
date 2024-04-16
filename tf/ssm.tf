resource "aws_ssm_parameter" "public_api_domain" {
  name        = "/application/PublicApiDomain"
  description = "Emprovise Public API domain name"
  type        = "String"
  value       = aws_api_gateway_domain_name.api_domain.domain_name
}

resource "aws_ssm_parameter" "application_subnets" {
  name        = "/application/vpc_subnets"
  description = "The ID list of application subnets"
  type        = "StringList"
  value       = join(",", module.app_vpc.private_subnets)
}

resource "aws_ssm_parameter" "vpc_sg" {
  name        = "/application/vpc_security_groups"
  description = "ID of VPC Security Group"
  type        = "StringList"
  value       = module.app_vpc.default_security_group_id
}

# SSM Parameter for serverless framework to find the bucket name
resource "aws_ssm_parameter" "serverless_state_bucket_name" {
  name        = "serverless-s3-bucket"
  description = "Serverless deployment bucket name"
  type        = "String"
  value       = aws_s3_bucket.sls_bucket.bucket
}
