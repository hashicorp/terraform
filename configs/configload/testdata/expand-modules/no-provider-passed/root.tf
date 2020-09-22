provider "aws" {
  alias  = "usw2"
  region = "us-west-2"
}
module "child" {
  count = 1
  source = "./child"
  # To make this test fail, add a valid providers {} block passing "aws" to the child
}
