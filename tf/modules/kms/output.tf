output "kms_key_arn" {
  value = aws_kms_key.this.arn
}

output "kms_key_id" {
  value = aws_kms_key.this.key_id
}

output "kms_alias_name" {
  value = aws_kms_alias.kms_key.name
}