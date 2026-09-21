resource "aws_instance" "child" {
}

module "grandchild" {
  source = "./grandchild"
}
