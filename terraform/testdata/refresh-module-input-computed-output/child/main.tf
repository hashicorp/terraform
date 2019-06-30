variable "input" {
    type = list(string)
}

resource "aws_instance" "foo" {
    foo = "${var.input}"
}

output "foo" {
    value = "${aws_instance.foo.foo}"
}
