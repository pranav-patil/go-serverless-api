variable "name" {
  description = "Name of KMS key"
  type        = string
  default     = ""
}

variable "description" {
  description = "KMS key description"
  type        = string
}

variable "account_id" {
  description = "AWS account ID"
  type        = string
  default     = ""
}

variable "key_usage" {
  description = "See https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_key#key_usage"
  type        = string
  default     = "ENCRYPT_DECRYPT"
}

variable "customer_master_key_spec" {
  description = "See https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_key#customer_master_key_spec"
  type        = string
  default     = "SYMMETRIC_DEFAULT"
}

variable "deletion_window_in_days" {
  description = "See https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_key#deletion_window_in_days"
  type        = number
  default     = 30
}

variable "is_enabled" {
  description = "See https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_key#is_enabled"
  type        = bool
  default     = true
}

variable "enable_key_rotation" {
  description = "See https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_key#enable_key_rotation"
  type        = bool
  default     = true
}

variable "key_users" {
  description = "IAM users/roles allowed to use the KMS key"
  type        = list(string)
  default     = []
}

variable "policy_documents" {
  description = "List of extra policy documents.  Elements must be in JSON format."
  type        = list(string)
  default     = []

  validation {
    condition     = alltrue([for x in var.policy_documents : can(jsondecode(x))])
    error_message = "All elements must be in JSON format."
  }
}

variable "tags" {
  type    = map(string)
  default = null
}

variable "multi_region" {
  description = "See https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_key#multi_region"
  type        = bool
  default     = false
}

variable "primary_key_arn" {
  type        = string
  description = "ARN of primary KMS key."
  default     = ""
}
