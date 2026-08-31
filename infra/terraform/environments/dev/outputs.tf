output "cluster_name" {
  description = "EKS cluster name."
  value       = module.eks.cluster_name
}

output "cluster_endpoint" {
  description = "EKS API endpoint."
  value       = module.eks.cluster_endpoint
  sensitive   = true
}

output "configure_kubectl" {
  description = "Command that writes the kubeconfig context."
  value       = "aws eks update-kubeconfig --region ${var.aws_region} --name ${module.eks.cluster_name}"
}

output "vpc_id" {
  description = "VPC ID."
  value       = module.vpc.vpc_id
}

output "private_subnet_ids" {
  description = "Private node subnet IDs."
  value       = module.vpc.private_subnets
}

output "public_subnet_ids" {
  description = "Public subnet IDs used by internet-facing load balancers and NAT gateways."
  value       = module.vpc.public_subnets
}

output "control_plane_subnet_ids" {
  description = "Isolated intra subnet IDs used by EKS control-plane ENIs."
  value       = module.vpc.intra_subnets
}

output "internet_gateway_id" {
  description = "VPC internet gateway ID."
  value       = module.vpc.igw_id
}

output "nat_gateway_ids" {
  description = "NAT gateway IDs used for private-subnet egress."
  value       = module.vpc.natgw_ids
}

output "route_table_ids" {
  description = "Route tables grouped by subnet tier."
  value = {
    public  = module.vpc.public_route_table_ids
    private = module.vpc.private_route_table_ids
    intra   = module.vpc.intra_route_table_ids
  }
}

output "security_group_ids" {
  description = "EKS cluster and node security-group IDs."
  value = {
    cluster = module.eks.cluster_security_group_id
    nodes   = module.eks.node_security_group_id
  }
}

output "iam_role_arns" {
  description = "Human-readable inventory of the major EKS and workload IAM roles."
  value = {
    cluster                      = module.eks.cluster_iam_role_arn
    vpc_cni                      = module.vpc_cni_pod_identity.iam_role_arn
    ebs_csi                      = module.ebs_csi_pod_identity.iam_role_arn
    aws_load_balancer_controller = module.aws_load_balancer_controller_pod_identity.iam_role_arn
    external_secrets             = module.external_secrets_pod_identity.iam_role_arn
  }
}

output "account_id" {
  description = "AWS account receiving the resources."
  value       = data.aws_caller_identity.current.account_id
}
