resource "aws_instance" "foo" {
}

output "id" {
  value = "${aws_instance.foo.id}"
}
