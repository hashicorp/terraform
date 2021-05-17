resource "aws_instance" "a" {
  require_new = "new"
}

output "ids" {
  value = [aws_instance.a.id]
}
