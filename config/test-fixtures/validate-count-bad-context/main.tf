resource "aws_instance" "foo" {
}

output "notgood" {
  value = "${count.index}"
}
