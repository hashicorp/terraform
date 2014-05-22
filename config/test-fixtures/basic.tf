variable "foo" {
    default = "bar";
    description = "bar";
}

resource "aws_security_group" "firewall" {
}

resource aws_instance "web" {
    ami = "ami-123456"
    security_groups = [
        "foo",
        "${aws_security_group.firewall.foo}"
    ]
}
