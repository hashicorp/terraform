variable "pass" {
}

variable "value" {
}

resource "aws_instance" "foo" {
    num = "2"
    compute = "value"
    compute_value = "${var.value}"
}

resource "aws_instance" "bar" {
    connection {
        host = "localhost"
        type = "telnet"
    }

    provisioner "shell" {
        foo = "${aws_instance.foo.value}"
        connection {
            host = "localhost"
            type = "telnet"
            user = "superuser"
            port = 2222
            password = "${var.pass}"
        }
    }
}
