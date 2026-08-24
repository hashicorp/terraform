
variable "input" {
  type = number
}

resource "test_resource" "my_resource" {
  apply_fail = true
}

output "output" {
  value = var.input
}
