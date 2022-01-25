variable "value" {}

resource "aws_instance" "bar" {
    foo = "${var.value}"
}
