variable "account_id" {
  description = "AWS account ID"
  type        = string
}

variable "user" {
  description = "User making infrastructure changes"
  type        = string
}

variable "stack" {
  description = "Stack being deployed to"
  type        = string
}

variable "stage" {
  description = "Stage being deployed to"
  type        = string
}

variable "region" {
  description = "AWS region"
  type        = string
}

variable "s3_suffix" {
  description = "Suffix for deploying S3 bucket"
  type        = string
}

variable "api_domain" {
  description = "Custom domain name for backend API Gateway"
  type        = string
}

variable "app_regions" {
  description = "Cloud One regions"
  type        = list(string)
  default     = []
}

variable "app_vpc_cidr" {
  description = "The CIDR of Application VPC"
  type        = string
  default     = ""
}

variable "intra_nat_cidr" {
  description = "The CIDR of internal subnet in Application VPC"
  type        = string
  default     = ""
}

variable "slack_channels" {
  description = "Service slack channel ID(s)"
  type        = map(string)
  default     = {}
}

variable "sns_regions" {
  description = "SNS regions Chatbot subscribes"
  type        = list(string)
}


