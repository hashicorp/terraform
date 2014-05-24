variable "foo" {
    default = "bar";
    description = "bar";
}

resource "aws_security_group" "firewall" {}

resource "aws_instance" "web" {
    ami = "${var.foo}"
    security_groups = [
        "foo",
        "${aws_security_group.firewall.foo}"
    ]
}

resource "aws_load_balancer" "weblb" {
    members = "${aws_instance.web.id_list}"
}
