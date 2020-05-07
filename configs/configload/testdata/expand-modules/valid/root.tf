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
  count = 1
  source = "./child-with-alias"
  providers = {
    aws.east = aws.east
  }
}