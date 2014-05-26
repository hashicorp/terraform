variable "foo" {
    default = "bar";
    description = "bar";
}

provider "aws" {
  access_key = "foo";
  secret_key = "bar";
}

provider "do" {
  api_key = "${var.foo}";
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
