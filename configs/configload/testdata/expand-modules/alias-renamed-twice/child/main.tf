provider "aws" {
  alias = "child_west"
}

module "child2" {
  source = "../child2"
  providers = {
    aws.child2_west = aws.child_west
  }
}
