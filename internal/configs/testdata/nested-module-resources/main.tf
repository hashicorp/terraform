resource "aws_instance" "root" {
}

module "child" {
  source = "./child"
}
