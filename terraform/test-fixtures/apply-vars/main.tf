variable "amis" {
    default = {
        us-east-1 = "foo"
        us-west-2 = "foo"
    }
}

variable "test_list" {
    type = "list"
}

variable "test_map" {
    type = "map"
}

variable "bar" {
    default = "baz"
}

variable "foo" {}

resource "aws_instance" "foo" {
    num = "2"
    bar = "${var.bar}"
    list = "${join(",", var.test_list)}"
    map = "${join(",", keys(var.test_map))}"
}

resource "aws_instance" "bar" {
    foo = "${var.foo}"
    bar = "${lookup(var.amis, var.foo)}"
    baz = "${var.amis["us-east-1"]}"
}
