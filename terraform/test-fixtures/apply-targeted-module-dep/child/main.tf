resource "aws_instance" "mod" { }

output "output" {
  value = "${aws_instance.mod.id}"
}
