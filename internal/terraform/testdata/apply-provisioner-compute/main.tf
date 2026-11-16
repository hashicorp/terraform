variable "value" {}

resource "aws_instance" "foo" {
    num = "2"
    compute = "value"
    compute_value = "${var.value}"
}

resource "aws_instance" "bar" {
    provisioner "shell" {
        command = "${aws_instance.foo.value}"
    }
}
