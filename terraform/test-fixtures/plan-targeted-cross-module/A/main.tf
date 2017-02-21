resource "aws_instance" "foo" {
    foo = "bar"
}

output "value" { value = "${aws_instance.foo.id}" }
