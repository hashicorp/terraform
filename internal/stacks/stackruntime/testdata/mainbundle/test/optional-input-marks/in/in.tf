variable "sensitive_input" {
  type      = string
  default   = "default-sensitive"
  sensitive = true
}

variable "ephemeral_input" {
  type      = string
  default   = "default-secret"
  ephemeral = true
}

resource "testing_write_only_resource" "resource" {
  id         = "optional-test"
  write_only = var.ephemeral_input
}
