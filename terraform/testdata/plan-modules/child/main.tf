resource "aws_instance" "foo" {
  count = 2
  num = 2
}

output "out" {
  value = aws_instance.foo[0].id
}
