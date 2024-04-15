#-----------------------------------------------------------------------------
# Serverless CloudFormation Execution Role
# - Assumed by CloudFormation to create the stack from the CloudFormation template
#-----------------------------------------------------------------------------

resource "aws_iam_role" "serverless_cloudformation_exec_role" {
  name        = "APPRoleForServerlessDeploy"
  description = "Assumed by Cloud Formation to deploy a serverless stack"
  assume_role_policy   = data.aws_iam_policy_document.cloudformation_assume_role.json
}

data "aws_iam_policy_document" "cloudformation_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["cloudformation.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy" "serverless_cloudformation_exec" {
  name   = "ServerlessCloudFormationExecutionPolicy"
  role   = aws_iam_role.serverless_cloudformation_exec_role.name
  policy = data.aws_iam_policy_document.serverless_cloudformation_exec.json
}

data "aws_iam_policy_document" "serverless_cloudformation_exec" {
  # Allow resources in the CloudFormation Template to be created
  # tfsec:ignore:aws-iam-no-policy-wildcards TIP-82345 apigateways
  statement {
    actions = [
      "apigateway:*",
      "cloudwatch:*",
      "events:*",
      "lambda:*",
      "logs:*",
      "s3:*",
      "sts:*",
      "sns:*",
      "ssm:*",
      "ec2:Describe*",
      "ec2:Get*",
      "ec2:List*",
      "ec2:CreateNetworkInterface",
      "ec2:DeleteNetworkInterface",
      "sqs:*",
      "iot:ListTagsForResource",
      "iot:UntagResource",
      "iot:TagResource",
      "iot:ListAuditTasks",
      "iot:StartOnDemandAuditTask",
      "iot:DescribeAuditTask",
      "iot:*TopicRule",
      "iot:*DomainConfiguration*",
      "iot:*FleetMetric*",
      "iot:*ScheduledAudit*",
      "iot:*AccountAuditConfiguration",
      "wafv2:*",
      "scheduler:GetSchedule",
      "scheduler:GetScheduleGroup",
      "scheduler:CreateScheduleGroup",
      "scheduler:DeleteScheduleGroup",
      "scheduler:TagResource",
      "scheduler:UntagResource",
      "scheduler:ListTagsForResource",
      "scheduler:ListSchedules",
      "scheduler:DeleteSchedule",
      "tag:GetResources",
      "states:*"
    ]
    resources = ["*"] # tfsec:ignore:aws-iam-no-policy-wildcards
  }

  # Allow artifacts to be accessed from the Serverless S3 Bucket (e.g. Lambda code)
  # Allow environment variables to be encrypted
  statement {
    actions = [
      "kms:Decrypt",
      "kms:GenerateDataKey",
      "kms:ListAliases",
      "kms:CreateGrant",
      "kms:Encrypt",
    ]
    resources = ["*"] # tfsec:ignore:aws-iam-no-policy-wildcards
  }
}

output "cloudformation_exec_role_name" {
  value = aws_iam_role.serverless_cloudformation_exec_role.name
}
