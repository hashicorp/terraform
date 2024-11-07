variable "value" {
  type = string
}

variable "id" {
  type = string
}

resource "test_resource" "managed" {
  id = var.id
  value = var.value
}
