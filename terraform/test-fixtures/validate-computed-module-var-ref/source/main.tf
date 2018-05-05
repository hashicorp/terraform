resource "aws_instance" "source" {}

output "sourceout" {
  value = "${aws_instance.source.id}"
}
