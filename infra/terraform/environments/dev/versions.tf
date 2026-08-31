terraform {
  required_version = "~> 1.16.0"

  # Account-specific backend values live in ignored backend.hcl files.
  backend "s3" {}

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.52"
    }
  }
}
