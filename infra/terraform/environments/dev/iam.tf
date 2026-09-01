module "vpc_cni_pod_identity" {
  source  = "terraform-aws-modules/eks-pod-identity/aws"
  version = "2.9.0"

  name = "${var.cluster_name}-vpc-cni"

  attach_aws_vpc_cni_policy = true
  aws_vpc_cni_enable_ipv4   = true

  # The EKS add-on owns this association so its lifecycle follows the add-on.
  associations = {}
  tags         = local.tags
}

module "ebs_csi_pod_identity" {
  source  = "terraform-aws-modules/eks-pod-identity/aws"
  version = "2.9.0"

  name                      = "${var.cluster_name}-ebs-csi"
  attach_aws_ebs_csi_policy = true

  # The EKS add-on owns this association so its lifecycle follows the add-on.
  associations = {}
  tags         = local.tags
}

module "aws_load_balancer_controller_pod_identity" {
  source  = "terraform-aws-modules/eks-pod-identity/aws"
  version = "2.9.0"

  name                            = "${var.cluster_name}-aws-lbc"
  attach_aws_lb_controller_policy = true
  additional_policy_arns = local.public_delivery_enabled ? {
    cognito = aws_iam_policy.aws_load_balancer_controller_cognito[0].arn
  } : {}

  associations = {
    controller = {
      cluster_name    = module.eks.cluster_name
      namespace       = "kube-system"
      service_account = "aws-load-balancer-controller"
    }
  }

  tags = local.tags
}

data "aws_iam_policy_document" "aws_load_balancer_controller_cognito" {
  count = local.public_delivery_enabled ? 1 : 0

  statement {
    actions   = ["cognito-idp:DescribeUserPoolClient"]
    resources = [aws_cognito_user_pool.public[0].arn]
  }
}

resource "aws_iam_policy" "aws_load_balancer_controller_cognito" {
  count = local.public_delivery_enabled ? 1 : 0

  name_prefix = "${var.cluster_name}-aws-lbc-cognito-"
  description = "Allow the AWS Load Balancer Controller to configure Cognito authentication"
  policy      = data.aws_iam_policy_document.aws_load_balancer_controller_cognito[0].json
  tags        = local.tags
}

module "external_dns_pod_identity" {
  count = local.public_delivery_enabled ? 1 : 0

  source  = "terraform-aws-modules/eks-pod-identity/aws"
  version = "2.9.0"

  name                          = "${var.cluster_name}-external-dns"
  attach_external_dns_policy    = true
  external_dns_hosted_zone_arns = [aws_route53_zone.public[0].arn]

  associations = {
    controller = {
      cluster_name    = module.eks.cluster_name
      namespace       = "external-dns"
      service_account = "external-dns"
    }
  }

  tags = local.tags
}

module "external_secrets_pod_identity" {
  source  = "terraform-aws-modules/eks-pod-identity/aws"
  version = "2.9.0"

  name                           = "${var.cluster_name}-external-secrets"
  attach_external_secrets_policy = true
  external_secrets_secrets_manager_arns = [
    "arn:${data.aws_partition.current.partition}:secretsmanager:${var.aws_region}:${data.aws_caller_identity.current.account_id}:secret:${var.external_secret_prefix}*"
  ]

  associations = {
    controller = {
      cluster_name    = module.eks.cluster_name
      namespace       = "external-secrets"
      service_account = "external-secrets"
    }
  }

  tags = local.tags
}
