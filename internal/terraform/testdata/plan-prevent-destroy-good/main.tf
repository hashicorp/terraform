resource "aws_instance" "foo" {
  lifecycle {
    prevent_destroy = true
  }
}
