resource "aws_instance" "foo" {
  provisioner "shell" {}
}

module "child" {
  source = "./child"
}
