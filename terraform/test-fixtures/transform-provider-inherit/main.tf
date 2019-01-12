provider "aws" {
  alias = "foo"
  value = "config"
}

module "child" {
  source = "./child"
  providers = {
    aws.bar = aws.foo
  }
}
