variable "foo" {
    default = "bar"
    description = "bar"
}

provider "aws" {
  access_key = "foo"
  secret_key = "bar"
}

resource "aws_instance" "db" {
    security_groups = "${aws_security_group.firewall.*.id}"
}

output "web_ip" {
    value = "${aws_instance.web.private_ip}"
}
