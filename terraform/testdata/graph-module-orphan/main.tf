provider "aws" {}

resource "aws_security_group" "firewall" {}

resource "aws_instance" "web" {
    security_groups = [
        "foo",
        "${aws_security_group.firewall.foo}"
    ]
}
