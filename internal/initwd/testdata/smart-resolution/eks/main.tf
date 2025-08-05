# Local eks module that calls registry modules
# Mimics the real eks module structure with pod identity modules

# Registry modules called from within this local module
# These should also go through smart resolution

module "cert_manager_pod_identity" {
  source = "terraform-aws-modules/eks-pod-identity/aws"
  # NO VERSION CONSTRAINT - smart resolution must find compatible version
}

module "aws_ebs_csi_pod_identity" {
  source = "terraform-aws-modules/eks-pod-identity/aws"
  # NO VERSION CONSTRAINT - smart resolution must find compatible version
}

module "aws_lb_controller_pod_identity" {
  source = "terraform-aws-modules/eks-pod-identity/aws"
  # NO VERSION CONSTRAINT - smart resolution must find compatible version
}

module "default_data_encryption" {
  source  = "terraform-aws-modules/kms/aws"
  version = ">= 3.0"
}

# This module has a submodule path
module "karpenter" {
  source  = "terraform-aws-modules/eks/aws//modules/karpenter"
  version = ">= 20.0"
}