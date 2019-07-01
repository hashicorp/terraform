variable "foo" {}

module "child" {
    input = "${var.foo}"
    source = "./child"
}

resource "aws_instance" "bar" {
    foo = "2"
}
