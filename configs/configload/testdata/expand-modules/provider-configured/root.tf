provider "aws" {
  region = "us-west-2"
}

module "child" {
  count = 1
  source = "./child"
  providers = {
    aws = aws.w2
  }
}
