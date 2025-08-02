module "example" {
  source = "terraform-aws-modules/vpc/aws"
  version = "not-a-version"  # ERROR: Invalid module version
  hashes = [
    "h1:test-hash",
  ]
}