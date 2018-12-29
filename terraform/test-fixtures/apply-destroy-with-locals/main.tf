locals {
  name = "test-${aws_instance.foo.id}"
}
resource "aws_instance" "foo" {}

output "name" {
  value = "${local.name}"
}
