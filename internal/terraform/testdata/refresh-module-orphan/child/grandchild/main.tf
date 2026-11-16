resource "aws_instance" "baz" {}

output "id" { value = "${aws_instance.baz.id}" }
