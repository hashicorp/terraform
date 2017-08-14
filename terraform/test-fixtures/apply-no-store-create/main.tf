resource "aws_instance" "foo" {
  required_field = "set"

  lifecycle {
    no_store = ["required_field"]
  }
}
