terraform {
  required_version = "~> 1.16.0"

  # Bootstrap starts with `terraform init -backend=false`; after the bucket
  # exists, operators migrate this state with the documented backend settings.
  backend "s3" {}

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.52"
    }
  }
}
