variable "ami" {
    default = "foo"
}

resource "aws_instance" "bar" {
    foo = "${var.ami}"
}
