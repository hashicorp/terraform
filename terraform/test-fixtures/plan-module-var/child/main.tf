resource "aws_instance" "foo" {
    num = "2"
}

output "num" {
    value = "${aws_instance.foo.num}"
}
