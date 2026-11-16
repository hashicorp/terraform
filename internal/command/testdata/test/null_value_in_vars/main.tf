
variable "null_input" {
  type = string
  default = null
}

variable "interesting_input" {
  type = string
  nullable = false
}

resource "test_resource" "resource" {
  value = var.interesting_input
}

output "null_output" {
  value = var.null_input
}
