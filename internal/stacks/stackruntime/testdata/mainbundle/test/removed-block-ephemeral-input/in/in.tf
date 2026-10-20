variable "secret" {
  type      = string
  ephemeral = true
}

resource "testing_write_only_resource" "resource" {
  id         = "removed-block"
  write_only = var.secret
}
