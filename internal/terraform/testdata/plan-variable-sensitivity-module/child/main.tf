variable "foo" {
  type = string
}

// "bar" is defined as sensitive by both the parent and the child
variable "bar" {
  sensitive = true
}

resource "aws_instance" "foo" {
  foo   = var.foo
  value = var.bar
}
