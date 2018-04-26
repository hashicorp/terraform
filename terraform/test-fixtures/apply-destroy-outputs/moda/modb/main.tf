resource "aws_instance" "foo" {
  id = "foo"
}

output "foo" {
  value = "${aws_instance.foo.id}"
}
