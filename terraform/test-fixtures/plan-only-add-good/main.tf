resource "aws_instance" "foo" {
  lifecycle {
     only_add = true
  }
}
