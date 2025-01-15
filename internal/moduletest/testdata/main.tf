
variable "input" {
  type = string
}


resource "foo_resource" "a" {
  value = var.input
}

resource "bar_resource" "c" {}

output "name" {
  value = foo_resource.a.value
}