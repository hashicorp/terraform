variable "foo" {}

variable "bar" {}

resource "aws_instance" "foo" {
  ami           = "${var.foo}"
  instance_type = "${var.bar}"

  lifecycle {
    ignore_changes = ["*"]
  }
}
