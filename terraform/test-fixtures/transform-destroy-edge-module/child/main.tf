resource "aws_instance" "b" {
    value = "foo"
}

output "output" { value = "${aws_instance.b.value}" }
