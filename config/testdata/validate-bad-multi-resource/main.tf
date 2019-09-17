resource "aws_instance" "web" {
  count = 5
}

output "ip" {
  value = "${aws_instance.web.id}"
}
