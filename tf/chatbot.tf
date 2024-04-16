resource "aws_iam_role_policy_attachment" "chatbot_execution_role_read_only_access" {
  role       = aws_iam_role.chatbot_execution_role.name
  policy_arn = "arn:aws:iam::aws:policy/ReadOnlyAccess"
}

resource "awscc_chatbot_slack_channel_configuration" "slack" {
  # TODO: replace vcs_public_api with {} once public api fix false alarm
  for_each = var.slack_channels

  configuration_name = each.key
  iam_role_arn       = aws_iam_role.chatbot_execution_role.arn
  slack_channel_id   = each.value
  slack_workspace_id = "T06V57ZGZ17"
  sns_topic_arns     = [for region in var.sns_regions : "arn:aws:sns:${region}:${var.account_id}:${each.key}"]
}
