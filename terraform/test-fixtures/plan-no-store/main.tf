variable "foo" {}

resource "aws_instance" "foo" {
  ami = "${var.foo}"

  lifecycle {
    no_store = ["ami"]
  }
}
