# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/aws" {
  version     = "3.74.0"
  constraints = ">= 3.29.0"
  hashes = [
    "h1:test-provider-hash-1",
    "h1:test-provider-hash-2",
  ]
}

module "module.local_example" {
  source = "./modules/local-example"
  hashes = [
    "h1:test-module-hash-1",
    "h1:test-module-hash-2",
  ]
}

module "module.registry_example" {
  source = "registry.terraform.io/terraform-aws-modules/vpc/aws"
  hashes = [
    "h1:test-module-hash-3",
  ]
}