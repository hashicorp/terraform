variable "input" {}

resource "aws_instance" "bar" {
    foo = "${var.input}"
}
