// this is the provider that should actually be used by orphaned resources
provider "aws" {
  alias = "bar"
}

module "mod" {
  source = "./mod"
  providers = {
    "aws.foo" = "aws.bar"
  }
}
