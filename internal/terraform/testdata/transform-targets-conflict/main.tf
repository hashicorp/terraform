resource "aws_vpc" "me" {}

module "child" {
  source = "./child"
}
