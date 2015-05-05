resource "aws_instance" "a" {}

output "a_output" {
    value = "${aws_instance.a.id}"
}
