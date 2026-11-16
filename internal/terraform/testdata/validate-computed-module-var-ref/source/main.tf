resource "aws_instance" "source" {
  attr = "foo"
}

output "sourceout" {
  value = "${aws_instance.source.attr}"
}
