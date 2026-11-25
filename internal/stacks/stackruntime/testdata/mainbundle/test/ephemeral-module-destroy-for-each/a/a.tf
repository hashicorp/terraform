variable "name" {
  type = string
}

resource "testing_resource" "a" {
  id    = "a-${var.name}"
  value = "a-value-${var.name}"
}

output "value" {
  value = testing_resource.a.value
}
