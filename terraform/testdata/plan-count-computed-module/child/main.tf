variable "value" {}

resource "aws_instance" "bar" {
    count = "${var.value}"
}
