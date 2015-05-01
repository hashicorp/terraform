variable "var" {}

resource "aws_instance" "child" {
    value = "${var.var}"
}
