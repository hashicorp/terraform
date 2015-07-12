variable "value" {}

resource "aws_instance" "foo" {
    num = "2"
    compute = "dynamical"
    compute_value = "${var.value}"
}

resource "aws_instance" "bar" {
    provisioner "shell" {
        foo = "${aws_instance.foo.dynamical}"
    }
}
