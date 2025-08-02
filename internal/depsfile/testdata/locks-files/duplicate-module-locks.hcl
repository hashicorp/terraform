module "example" {
  source = "terraform-aws-modules/vpc/aws"
  version = "1.0.0"
  hashes = [
    "h1:test-hash-1",
  ]
}

module "example" {  # ERROR: Duplicate module lock
  source = "terraform-aws-modules/vpc/aws"
  version = "2.0.0"
  hashes = [
    "h1:test-hash-2",
  ]
}