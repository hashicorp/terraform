resource "aws_instance" "foo" {}

module "child" {
    source = "./child"
}
