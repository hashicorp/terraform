resource "aws_instance" "a" {
  id = "a"
}

output "a_output" {
  value = "${aws_instance.a.id}"
}
