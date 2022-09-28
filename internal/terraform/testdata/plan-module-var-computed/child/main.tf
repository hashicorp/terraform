resource "aws_instance" "foo" {
    compute = "foo"
}

output "num" {
    value = "${aws_instance.foo.foo}"
}
