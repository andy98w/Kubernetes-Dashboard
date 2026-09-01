locals {
  azs = slice(data.aws_availability_zones.available.names, 0, 3)

  tags = merge(
    {
      Project     = "KubeVista"
      Environment = var.environment
      ManagedBy   = "Terraform"
      Repository  = "andy98w/Kubernetes-Dashboard"
      CostCenter  = "portfolio"
    },
    var.expires_at == null ? {} : { ExpiresAt = var.expires_at },
    var.tags,
  )
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "6.6.1"

  name = "${var.cluster_name}-vpc"
  cidr = var.vpc_cidr
  azs  = local.azs

  private_subnets = [for index, _ in local.azs : cidrsubnet(var.vpc_cidr, 4, index)]
  public_subnets  = [for index, _ in local.azs : cidrsubnet(var.vpc_cidr, 4, index + 4)]
  intra_subnets   = [for index, _ in local.azs : cidrsubnet(var.vpc_cidr, 4, index + 8)]

  enable_nat_gateway     = true
  single_nat_gateway     = var.single_nat_gateway
  one_nat_gateway_per_az = !var.single_nat_gateway
  enable_dns_hostnames   = true
  enable_dns_support     = true

  enable_flow_log                                 = true
  create_flow_log_cloudwatch_iam_role             = true
  create_flow_log_cloudwatch_log_group            = true
  flow_log_cloudwatch_log_group_retention_in_days = 30
  flow_log_max_aggregation_interval               = 60

  public_subnet_tags = {
    "kubernetes.io/role/elb" = 1
  }

  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" = 1
    "karpenter.sh/discovery"          = var.cluster_name
  }

  tags = local.tags
}

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "21.24.2"

  name               = var.cluster_name
  kubernetes_version = var.kubernetes_version

  authentication_mode                      = "API"
  enable_cluster_creator_admin_permissions = false
  enable_irsa                              = false

  endpoint_private_access                = true
  endpoint_public_access                 = length(var.public_access_cidrs) > 0
  endpoint_public_access_cidrs           = var.public_access_cidrs
  enabled_log_types                      = ["api", "audit", "authenticator", "controllerManager", "scheduler"]
  cloudwatch_log_group_retention_in_days = 30

  encryption_config = {
    resources = ["secrets"]
  }

  addons = {
    coredns = {}
    eks-pod-identity-agent = {
      before_compute = true
    }
    aws-ebs-csi-driver = {
      pod_identity_association = [{
        role_arn        = module.ebs_csi_pod_identity.iam_role_arn
        service_account = "ebs-csi-controller-sa"
      }]
    }
    kube-proxy = {}
    vpc-cni = {
      before_compute = true
      most_recent    = true
      configuration_values = jsonencode({
        enableNetworkPolicy = "true"
      })
      pod_identity_association = [{
        role_arn        = module.vpc_cni_pod_identity.iam_role_arn
        service_account = "aws-node"
      }]
    }
  }

  vpc_id                   = module.vpc.vpc_id
  subnet_ids               = module.vpc.private_subnets
  control_plane_subnet_ids = module.vpc.intra_subnets

  access_entries = var.admin_principal_arn == null ? {} : {
    platform-admin = {
      principal_arn = var.admin_principal_arn
      policy_associations = {
        cluster-admin = {
          policy_arn = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy"
          access_scope = {
            type = "cluster"
          }
        }
      }
    }
  }

  eks_managed_node_groups = {
    baseline = {
      ami_type       = "AL2023_x86_64_STANDARD"
      instance_types = var.node_instance_types
      capacity_type  = "ON_DEMAND"

      # VPC CNI receives its AWS permissions through EKS Pod Identity.
      # Keep those permissions off the broader EC2 node role.
      iam_role_attach_cni_policy = false

      min_size     = 2
      max_size     = 4
      desired_size = 2

      update_config = {
        max_unavailable_percentage = 33
      }

      metadata_options = {
        http_endpoint               = "enabled"
        http_tokens                 = "required"
        http_put_response_hop_limit = 1
        instance_metadata_tags      = "disabled"
      }

      labels = {
        workload = "general"
      }
    }
  }

  node_security_group_tags = {
    "karpenter.sh/discovery" = var.cluster_name
  }

  tags = local.tags
}

resource "aws_budgets_budget" "monthly" {
  name         = "${var.cluster_name}-monthly"
  budget_type  = "COST"
  limit_amount = tostring(var.monthly_budget_usd)
  limit_unit   = "USD"
  time_unit    = "MONTHLY"

  cost_filter {
    name   = "TagKeyValue"
    values = ["user:Project$KubeVista"]
  }

  dynamic "notification" {
    for_each = nonsensitive(var.budget_notification_email == null) ? {} : {
      actual-50 = {
        threshold         = 50
        notification_type = "ACTUAL"
      }
      forecasted-80 = {
        threshold         = 80
        notification_type = "FORECASTED"
      }
    }
    content {
      comparison_operator        = "GREATER_THAN"
      threshold                  = notification.value.threshold
      threshold_type             = "PERCENTAGE"
      notification_type          = notification.value.notification_type
      subscriber_email_addresses = [var.budget_notification_email]
    }
  }
}
