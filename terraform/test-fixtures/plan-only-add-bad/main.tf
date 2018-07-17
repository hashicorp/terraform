resource "aws_instance" "foo" {
  require_new = "yes"

  lifecycle {
    only_add = true
  }
}
