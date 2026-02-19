variable "foo" {
    default = "2"
}

resource "aws_instance" "foo" {
    foo = "foo"
    count = "${var.foo}"
}
