provider "aws" {
  foo = "bar"
}

module "child" {
  count = 1
  source = "./child"
  providers = {
    aws = aws
  }
}
