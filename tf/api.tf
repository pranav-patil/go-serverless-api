

resource "aws_api_gateway_domain_name" "api_domain" {
  regional_certificate_arn = aws_acm_certificate_validation.acm_cert_val.certificate_arn
  domain_name              = aws_acm_certificate.acm_cert.domain_name
  security_policy          = "TLS_1_2"
  tags                     = local.tags

  endpoint_configuration {
    types = ["REGIONAL"]
  }

  lifecycle {
    ignore_changes = [tags]
  }
}

resource "aws_acm_certificate" "acm_cert" {
  domain_name       = var.api_domain
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_zone" "primary" {
  name = "${var.api_domain}"
}

resource "aws_route53_record" "certval" {
  name    = tolist(aws_acm_certificate.acm_cert.domain_validation_options)[0].resource_record_name
  type    = tolist(aws_acm_certificate.acm_cert.domain_validation_options)[0].resource_record_type
  zone_id = aws_route53_zone.primary.zone_id
  records = [tolist(aws_acm_certificate.acm_cert.domain_validation_options)[0].resource_record_value]
  ttl     = 300
}

resource "aws_acm_certificate_validation" "acm_cert_val" {
  certificate_arn         = aws_acm_certificate.acm_cert.arn
  validation_record_fqdns = [aws_route53_record.certval.fqdn]

  timeouts {
    create = "10m"
  }
}