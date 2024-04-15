# [AWS097][ERROR] Resource 'xxx' a policy with KMS actions for all KMS keys.
# - This is OK since the policy document is attached to a specific aws_kms_key resource.
# - AWS example code for KSM key policies does the same (https://docs.aws.amazon.com/kms/latest/developerguide/key-policies.html)
# tfsec:ignore:AWS097
data "aws_iam_policy_document" "this" {
  policy_id               = var.name
  source_policy_documents = var.policy_documents

  statement {
    sid     = "Enable IAM User Permissions"
    actions = ["kms:*"]

    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::${var.account_id}:root"]
    }
    resources = ["*"]
  }

  statement {
    sid = "Allow access for Key Administrators"

    actions = [
      "kms:Create*",
      "kms:Describe*",
      "kms:Enable*",
      "kms:List*",
      "kms:Put*",
      "kms:Update*",
      "kms:Revoke*",
      "kms:Disable*",
      "kms:Get*",
      "kms:Delete*",
      "kms:TagResource",
      "kms:UntagResource",
      "kms:ScheduleKeyDeletion",
      "kms:CancelKeyDeletion"
    ]

    principals {
      type = "AWS"
      identifiers = ["arn:aws:iam::${var.account_id}:root"]
    }
    resources = ["*"]
  }

  statement {
    sid = "Allow use of the key"

    actions = [
      "kms:Encrypt",
      "kms:Decrypt",
      "kms:ReEncrypt*",
      "kms:GenerateDataKey*",
      "kms:DescribeKey"
    ]

    principals {
      type = "AWS"
      identifiers = ["arn:aws:iam::${var.account_id}:root"]
    }
    resources = ["*"]
  }

  statement {
    sid = "Allow attachment of persistent resources"

    actions = [
      "kms:CreateGrant",
      "kms:ListGrants",
      "kms:RevokeGrant"
    ]

    principals {
      type = "AWS"
      identifiers = ["arn:aws:iam::${var.account_id}:root"]
    }
    resources = ["*"]

    condition {
      test     = "Bool"
      variable = "kms:GrantIsForAWSResource"
      values   = [true]
    }
  }
}

# General kms creation
resource "aws_kms_key" "this" {
  description              = var.description
  key_usage                = var.key_usage
  customer_master_key_spec = var.customer_master_key_spec
  policy                   = data.aws_iam_policy_document.this.json
  deletion_window_in_days  = var.deletion_window_in_days
  is_enabled               = var.is_enabled
  enable_key_rotation      = var.enable_key_rotation
  tags                     = var.tags
  multi_region             = var.multi_region

  lifecycle {
    ignore_changes = [tags]
  }
}

resource "aws_kms_alias" "kms_key" {
  name          = format("alias/%v", var.name)
  target_key_id = aws_kms_key.this.key_id
}

resource "aws_ssm_parameter" "secrets_kms_key" {
  name        = "/kms/${var.name}"
  description = "${var.name} KMS key ARN"
  type        = "String"
  value       = aws_kms_key.this.arn
}
