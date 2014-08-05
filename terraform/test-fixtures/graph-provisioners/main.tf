variable "foo" {
    default = "bar"
    description = "bar"
}

provider "aws" {}

resource "aws_security_group" "firewall" {}

resource "aws_instance" "web" {
    ami = "${var.foo}"
    security_groups = [
        "foo",
        "${aws_security_group.firewall.foo}"
    ]
    provisioner "winrm" {
        cmd = "echo foo"
    }
    provisioner "winrm" {
        cmd = "echo bar"
    }
}

resource "aws_load_balancer" "weblb" {
    provisioner "shell" {
        cmd = "add ${aws_instance.web.id}"
        connection {
            type = "magic"
            user = "${aws_security_group.firewall.id}"
        }
    }
}
