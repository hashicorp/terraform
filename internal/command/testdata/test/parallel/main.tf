
variable "input" {
  type = string
}

variable "delay" {
  type = number
  default = 0
}

resource "test_resource" "foo" {
  create_wait_seconds = var.delay
  value = var.input
}

output "value" {
  value = test_resource.foo.value
}
