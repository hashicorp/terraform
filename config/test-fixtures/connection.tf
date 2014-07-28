resource "aws_instance" "web" {
    ami = "${var.foo}"
    security_groups = [
        "foo",
        "${aws_security_group.firewall.foo}"
    ]

    connection {
        type = "ssh"
        user = "root"
    }

    provisioner "shell" {
        path = "foo"
        connection {
            user = "nobody"
        }
    }

    provisioner "shell" {
        path = "bar"
    }
}
