locals {
  public_delivery_enabled = var.public_zone_name != null
  public_hostname         = var.public_zone_name
  cognito_domain_prefix   = "${var.cluster_name}-${data.aws_caller_identity.current.account_id}"
}

resource "aws_route53_zone" "public" {
  count = local.public_delivery_enabled ? 1 : 0

  name    = var.public_zone_name
  comment = "Delegated public zone for the KubeVista portfolio environment"

  tags = local.tags
}

resource "aws_route53_record" "public_caa" {
  count = local.public_delivery_enabled ? 1 : 0

  zone_id = aws_route53_zone.public[0].zone_id
  name    = var.public_zone_name
  type    = "CAA"
  ttl     = 300
  records = ["0 issue \"amazon.com\""]
}

resource "aws_acm_certificate" "public" {
  count = local.public_delivery_enabled ? 1 : 0

  domain_name       = local.public_hostname
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }

  tags = local.tags
}

resource "aws_route53_record" "certificate_validation" {
  for_each = local.public_delivery_enabled ? {
    for option in aws_acm_certificate.public[0].domain_validation_options :
    option.domain_name => {
      name   = option.resource_record_name
      record = option.resource_record_value
      type   = option.resource_record_type
    }
  } : {}

  zone_id = aws_route53_zone.public[0].zone_id
  name    = each.value.name
  type    = each.value.type
  ttl     = 60
  records = [each.value.record]

  allow_overwrite = true
}

resource "aws_cognito_user_pool" "public" {
  count = local.public_delivery_enabled ? 1 : 0

  name                     = "${var.cluster_name}-users"
  username_attributes      = ["email"]
  auto_verified_attributes = ["email"]
  mfa_configuration        = "ON"

  username_configuration {
    case_sensitive = false
  }

  software_token_mfa_configuration {
    enabled = true
  }

  password_policy {
    minimum_length                   = 14
    require_lowercase                = true
    require_numbers                  = true
    require_symbols                  = true
    require_uppercase                = true
    temporary_password_validity_days = 7
  }

  admin_create_user_config {
    allow_admin_create_user_only = true
  }

  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_email"
      priority = 1
    }
  }

  tags = local.tags
}

resource "aws_cognito_user_pool_client" "public" {
  count = local.public_delivery_enabled ? 1 : 0

  name         = "${var.cluster_name}-alb"
  user_pool_id = aws_cognito_user_pool.public[0].id

  generate_secret                      = true
  allowed_oauth_flows_user_pool_client = true
  allowed_oauth_flows                  = ["code"]
  allowed_oauth_scopes                 = ["openid", "email"]
  supported_identity_providers         = ["COGNITO"]
  callback_urls                        = ["https://${local.public_hostname}/oauth2/idpresponse"]
  logout_urls                          = ["https://${local.public_hostname}/"]

  prevent_user_existence_errors = "ENABLED"
  enable_token_revocation       = true

  access_token_validity  = 1
  id_token_validity      = 1
  refresh_token_validity = 1

  token_validity_units {
    access_token  = "hours"
    id_token      = "hours"
    refresh_token = "days"
  }
}

resource "aws_cognito_user_pool_domain" "public" {
  count = local.public_delivery_enabled ? 1 : 0

  domain       = local.cognito_domain_prefix
  user_pool_id = aws_cognito_user_pool.public[0].id
}

resource "aws_cognito_user" "admin" {
  count = local.public_delivery_enabled && var.public_admin_email != null ? 1 : 0

  user_pool_id = aws_cognito_user_pool.public[0].id
  username     = var.public_admin_email

  attributes = {
    email          = var.public_admin_email
    email_verified = true
  }

  desired_delivery_mediums = ["EMAIL"]
}
