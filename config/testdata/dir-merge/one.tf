variable "foo" {
    default = "bar"
    description = "bar"
}

resource "aws_instance" "db" {
    security_groups = "${aws_security_group.firewall.*.id}"
}
