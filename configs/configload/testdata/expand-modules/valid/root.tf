provider "aws" {
  region = "us-east-1"
  alias = "east"
}

module "child" {
  count = 1
  source = "./child"
  providers = {
    aws = aws.east
  }
}

module "child_with_alias" {
  for_each = toset(["a", "b"])
  source = "./child-with-alias"
  providers = {
    aws.east = aws.east
  }
}