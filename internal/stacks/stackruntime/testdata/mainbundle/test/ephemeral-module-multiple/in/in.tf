variable "id" {
  type = string
}

variable "input" {
  type      = string
  ephemeral = true
}

resource "testing_write_only_resource" "resource" {
  id         = var.id
  write_only = var.input
}
