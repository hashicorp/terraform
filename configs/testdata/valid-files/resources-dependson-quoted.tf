resource "aws_security_group" "firewall" {
}

resource "aws_instance" "web" {
  depends_on = [
    "aws_security_group.firewall",
  ]
}
