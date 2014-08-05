variable "amis" {
    default = {
        us-east-1 = "foo"
        us-west-2 = "foo"
    }
}

variable "bar" {
    default = "baz"
}

variable "foo" {}

resource "aws_instance" "foo" {
    num = "2"
    bar = "${var.bar}"
}

resource "aws_instance" "bar" {
    foo = "${var.foo}"
    bar = "${lookup(var.amis, var.foo)}"
    baz = "${var.amis.us-east-1}"
}
