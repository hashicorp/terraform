resource "aws_instance" "foo" {}

output "foo" {
    value = "${aws_instance.foo.value}"
}
