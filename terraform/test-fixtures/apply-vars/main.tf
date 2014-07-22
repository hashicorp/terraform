variable "amis" {
    default = {
        "us-east-1": "foo",
        "us-west-2": "foo",
    }
}

resource "aws_instance" "foo" {
    num = "2"
}

resource "aws_instance" "bar" {
    foo = "${var.foo}"
    bar = "${lookup(var.amis, var.foo)}"
}
