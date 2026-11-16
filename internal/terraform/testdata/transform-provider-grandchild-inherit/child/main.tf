provider "aws" {
  alias = "bar"
}

module "grandchild" {
  source = "./grandchild"
  providers = {
    aws.baz = aws.bar
  }
}
