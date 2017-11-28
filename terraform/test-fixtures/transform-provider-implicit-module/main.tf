provider "aws" {
  alias = "foo"
}

module "mod" {
  source = "./mod"
  providers = {
    "aws" = "aws.foo"
  }
}
