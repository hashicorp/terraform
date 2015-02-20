module "consul" {
    foo = "${aws_security_group.firewall.foo}"
    source = "./consul"
}

provider "aws" {}

resource "aws_security_group" "firewall" {}

resource "aws_instance" "web" {
    security_groups = [
        "foo",
        "${aws_security_group.firewall.foo}",
        "${module.consul.security_group}"
    ]
}
