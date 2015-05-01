variable "var" {}

resource "aws_instance" "child" {
    value = "${var.var}"
}

output "output" {
    value = "${aws_instance.child.value}"
}
