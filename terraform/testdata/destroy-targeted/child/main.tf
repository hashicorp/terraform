variable "in" {
}

resource "aws_instance" "b" {
  foo = var.in
}

output "out" {
  value = var.in
}
