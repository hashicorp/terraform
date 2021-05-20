resource "aws_instance" "a" { }

output "output" {
  value = "${aws_instance.a.id}"
}
