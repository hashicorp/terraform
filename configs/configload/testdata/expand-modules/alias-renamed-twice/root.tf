provider "aws" {
  alias  = "root_west"
  region = "us-west-2"
}

module "child" {
  count = 1
  source = "./child"
  providers = {
    aws.child_west = aws.root_west
  }
}
