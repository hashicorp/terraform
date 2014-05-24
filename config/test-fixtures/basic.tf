variable "foo" {
    default = "bar";
    description = "bar";
}

resource "aws_security_group" "firewall" {
}

resource aws_instance "web" {
    ami = "${var.foo}"
    security_groups = [
        "foo",
        "${aws_security_group.firewall.foo}"
    ]
}
