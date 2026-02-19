
variable "input_one" {
  type = string
}

variable "input_two" {
  type = string
}

resource "test_resource" "resource" {
  value = "${var.input_one} - ${var.input_two}"
}

output "response" {
  value = test_resource.resource.value
}
