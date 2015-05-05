variable "pass" {}
variable "value" {}

resource "aws_instance" "foo" {
    num = "2"
    compute = "dynamical"
    compute_value = "${var.value}"
}

resource "aws_instance" "bar" {
    connection {
        type = "telnet"
    }

    provisioner "shell" {
        foo = "${aws_instance.foo.dynamical}"
        connection {
            user = "superuser"
            port = 2222
            pass = "${var.pass}"
        }
    }
}
