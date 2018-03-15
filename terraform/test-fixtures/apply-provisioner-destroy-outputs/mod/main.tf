output "value" {
  value = "${aws_instance.baz.id}"
}

resource "aws_instance" "baz" {}
