resource "aws_sns_topic" "monitoring_topic" {
  for_each = var.slack_channels

  name              = each.key
  kms_master_key_id = module.kms_sns.kms_key_arn
  tags              = local.tags

  lifecycle {
    ignore_changes = [tags]
  }
}

resource "aws_sns_topic_subscription" "monitoring_subscription" {
  for_each = aws_sns_topic.monitoring_topic

  topic_arn = each.value.arn
  protocol  = "https"
  endpoint  = "https://global.sns-api.chatbot.amazonaws.com"
}
