provider "terraform.io/test/example" {
  version = "1.0.0"
  hashes = [
    "test:provider-hash-1",
  ]
}

module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
  version = "3.14.0"
  hashes = [
    "h1:abc123def456",
    "h1:xyz789uvw012",
  ]
}

module "subnet.private" {
  source = "git::https://github.com/example/terraform-modules.git//subnet"
  hashes = [
    "h1:git-module-hash",
  ]
}

module "local" {
  source = "./modules/local"
  hashes = [
    "h1:local-module-hash",
  ]
}