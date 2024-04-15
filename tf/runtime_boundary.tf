// Permissions boundary for resources
//
// Info:
// https://serverlessfirst.com/lambda-blast-radius-iam-permission-boundary/
// https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_boundaries.html
resource "aws_iam_policy" "runtime_boundary" {
  name        = "ApplicationRuntimeBoundary"
  description = "Permissions boundary for severless applications"
  path        = "/emprovise/"
  policy      = data.aws_iam_policy_document.runtime_boundary.json
}

data "aws_iam_policy_document" "runtime_boundary" {
  statement {
    # tfsec:ignore:aws-iam-no-policy-wildcards
    actions = [
      "execute-api:Invoke",
      "cloudwatch:*",
      "events:*",
      "kms:*",
      "lambda:*",
      "logs:*",
      "s3:*",
      "sns:*",
      "xray:*",
      "ssm:*",
      "secretsmanager:GetSecretValue",
      "elasticache:*",
      "ec2:AssignPrivateIpAddresses",
      "ec2:CreateNetworkInterface",
      "ec2:DeleteNetworkInterface",
      "ec2:DescribeNetworkInterfaces",
      "ec2:DescribeRouteTables",
      "ec2:DescribeInstances",
      "ec2:ReplaceRouteTableAssociation",
      "ec2:UnassignPrivateIpAddresses",
      "ec2:DescribeVpcs",
      "dynamodb:List*",
      "dynamodb:Describe*",
      "dynamodb:Get*",
      "dynamodb:*Item",
      "dynamodb:TagResource",
      "dynamodb:UntagResource",
      "dynamodb:PartiQL*",
      "sts:*",
      "sqs:*",
      "rds-db:connect"
    ]
    resources = ["*"] # tfsec:ignore:aws-iam-no-policy-wildcards
  }
}
