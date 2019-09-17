resource "aws_instance" "foo" {
  foo = "bar-$${baz}"
}
