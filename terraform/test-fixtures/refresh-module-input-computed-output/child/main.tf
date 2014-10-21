variable "input" {}

resource "aws_instance" "foo" {
    foo = "${var.input}"
}

output "foo" {
    value = "${aws_instance.foo.foo}"
}
