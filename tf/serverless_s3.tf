# tfsec:ignore:aws-s3-enable-bucket-logging (logging not needed for deployment bucket, CD pipeline contains deploy log)
resource "aws_s3_bucket" "sls_bucket" {
  bucket = "${var.stack}-sls-deployment-${var.s3_suffix}"
  lifecycle {
    ignore_changes = [server_side_encryption_configuration]
  }
}

resource "aws_s3_bucket_policy" "sls_bucket" {
  bucket = aws_s3_bucket.sls_bucket.id
  policy = jsonencode(
    {
      "Version" : "2012-10-17",
      "Statement" : [
        {
          "Action" : "s3:*",
          "Condition" : {
            "Bool" : {
              "aws:SecureTransport" : "false"
            }
          },
          "Effect" : "Deny",
          "Principal" : {
            "AWS" : "*"
          },
          "Resource" : "${aws_s3_bucket.sls_bucket.arn}/*"
        }
      ]
    }
  )
}

resource "aws_s3_bucket_public_access_block" "sls_bucket" {
  bucket = aws_s3_bucket.sls_bucket.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "sls_bucket" {
  bucket = aws_s3_bucket.sls_bucket.id

  rule {
    apply_server_side_encryption_by_default {
      kms_master_key_id = module.kms_s3.kms_key_arn
      sse_algorithm     = "aws:kms"
    }
  }
}

resource "aws_s3_bucket_versioning" "sls_bucket" {
  bucket = aws_s3_bucket.sls_bucket.id
  versioning_configuration {
    status = "Enabled"
  }
}
