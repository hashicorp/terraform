# Test fixture for smart module version resolution - real-world scenario
# Mimics the actual project structure where a local eks module calls registry modules

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.68"  # Specific AWS provider constraint
    }
  }
}

# Local module that contains registry module calls (like the real eks module)
module "eks" {
  source = "./eks"  # Local module, not subject to smart resolution
}

# Root-level registry modules that need smart resolution
module "nms_pod_identity" {
  source = "terraform-aws-modules/eks-pod-identity/aws"
  # NO VERSION CONSTRAINT - smart resolution must find compatible version
}

module "redis_encryption" {
  source  = "terraform-aws-modules/kms/aws"
  version = ">= 3.0"  # Version constraint that needs to work with AWS ~5.68
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = ">= 5.0"  # Recent VPC versions
}
