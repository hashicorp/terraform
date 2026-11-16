variable "id" {
  type     = string
  default  = null
  nullable = true # We'll generate an ID if none provided.
}

variable "input" {
  type = string
}

resource "testing_resource" "data" {
  id    = var.id
  value = var.input
}
