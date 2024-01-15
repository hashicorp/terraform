
variable "in" {
  type = string
}

resource "foo" "bleep" {
  arg = var.in
}

output "out" {
  value = foo.bleep.arg
}
