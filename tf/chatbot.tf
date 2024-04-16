
resource "awscc_chatbot_slack_channel_configuration" "slack" {
  for_each = var.slack_channels

  configuration_name = each.key
  iam_role_arn       = aws_iam_role.chatbot_execution_role.arn
  slack_channel_id   = each.value
  slack_workspace_id = "T06V57ZGZ17"
  sns_topic_arns     = [for region in var.sns_regions : "arn:aws:sns:${region}:${var.account_id}:${each.key}"]
}
