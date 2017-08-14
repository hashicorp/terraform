resource "aws_instance" "foo" {
  vars = "foo"

  lifecycle {
    no_store = ["vars"]
  }
}
