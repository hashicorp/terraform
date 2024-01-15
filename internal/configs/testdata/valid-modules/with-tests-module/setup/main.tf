variable "value" {
  type = string
}

variable "id" {
  type = string
}

resource "test_resource" "managed" {
  provider = setup
  id = var.id
  value = var.value
}
