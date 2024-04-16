resource "aws_iam_policy" "chatbot_permissions_boundary" {
  name        = "ChatbotPermissionBoundary"
  description = "Permissions boundary for chatbot service"

  policy = jsonencode(
    # tfsec:ignore:aws-iam-no-policy-wildcards
    {
      "Version" : "2012-10-17",
      "Statement" : [
        {
          "Effect" : "Allow",
          "Action" : [
            "cloudwatch:*",
            "logs:*",
            "sns:*"
          ]
          "Resource" : "*" # tfsec:ignore:AWS099
        }
      ]
    }
  )
}

resource "aws_iam_role" "chatbot_execution_role" {
  name                 = "SlackChatbotRole"
  description          = "Role used by AWS Chatbot to send alarms to slack channels"
  permissions_boundary = aws_iam_policy.chatbot_permissions_boundary.arn

  assume_role_policy = jsonencode(
    {
      "Version" : "2012-10-17",
      "Statement" : [
        {
          "Effect" : "Allow",
          "Principal" : {
            "Service" : "chatbot.amazonaws.com"
          },
          "Action" : "sts:AssumeRole"
        }
      ]
    }
  )

  tags = local.tags
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "aws_iam_role_policy_attachment" "chatbot_execution_role_read_only_access" {
  role       = aws_iam_role.chatbot_execution_role.name
  policy_arn = "arn:aws:iam::aws:policy/ReadOnlyAccess"
}
