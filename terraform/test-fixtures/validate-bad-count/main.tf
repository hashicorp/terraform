resource "aws_instance" "foo" {
  count = "${list}"
}
