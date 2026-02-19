
variable "input" {
  type = string
}


resource "foo_resource" "a" {
  value = var.input
}

output "output" {
  value = foo_resource.a.value
}
