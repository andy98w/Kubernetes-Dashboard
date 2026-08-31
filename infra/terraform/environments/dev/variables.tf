variable "aws_region" {
  description = "AWS region for the portfolio environment."
  type        = string
  default     = "us-west-2"
}

variable "environment" {
  description = "Lifecycle environment name."
  type        = string
  default     = "dev"
}

variable "cluster_name" {
  description = "EKS cluster name."
  type        = string
  default     = "kubevista-dev"
}

variable "kubernetes_version" {
  description = "EKS Kubernetes minor version."
  type        = string
  default     = "1.36"
}

variable "vpc_cidr" {
  description = "RFC1918 CIDR allocated to the environment."
  type        = string
  default     = "10.42.0.0/16"
}

variable "admin_principal_arn" {
  description = "IAM Identity Center or IAM role ARN granted cluster administrator access."
  type        = string
  default     = null
  nullable    = true

  validation {
    condition     = var.admin_principal_arn == null || can(regex("^arn:aws[a-z-]*:iam::[0-9]{12}:role/.+", var.admin_principal_arn))
    error_message = "admin_principal_arn must be an IAM role ARN, not an IAM user ARN."
  }
}

variable "public_access_cidrs" {
  description = "CIDRs allowed to reach the public API endpoint when it is enabled."
  type        = list(string)
  default     = []
}

variable "single_nat_gateway" {
  description = "Use one NAT gateway to reduce portfolio cost. Disable for one per AZ."
  type        = bool
  default     = true
}

variable "node_instance_types" {
  description = "Allowed instance types for the baseline managed node group."
  type        = list(string)
  default     = ["t3.large", "t3a.large"]
}

variable "monthly_budget_usd" {
  description = "Monthly AWS cost budget for the portfolio environment."
  type        = number
  default     = 100
}

variable "budget_notification_email" {
  description = "Optional email address for AWS Budget alerts."
  type        = string
  default     = null
  nullable    = true
}

variable "tags" {
  description = "Additional tags applied to all resources."
  type        = map(string)
  default     = {}
}

