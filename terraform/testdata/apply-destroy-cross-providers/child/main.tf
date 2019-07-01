variable "value" {}

resource "aws_vpc" "bar" {
    value = "${var.value}"
}
