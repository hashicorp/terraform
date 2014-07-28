resource "aws_instance" "web" {
    ami = "${var.foo}"
    security_groups = [
        "foo",
        "${aws_security_group.firewall.foo}"
    ]

    provisioner "shell" {
        path = "foo"
    }
}
