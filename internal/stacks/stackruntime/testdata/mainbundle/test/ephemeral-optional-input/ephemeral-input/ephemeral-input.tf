variable "input" {
  type      = string
  default   = "default-secret"
  ephemeral = true
}

resource "testing_write_only_resource" "resource" {
  id         = "optional-test"
  write_only = var.input
}
