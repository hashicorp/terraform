variable "secret" {
  type      = string
  ephemeral = true
  nullable  = false

  validation {
    condition     = length(var.secret) > 0
    error_message = "The secret must not be empty."
  }
}

resource "testing_write_only_resource" "resource" {
  id         = "removed-block"
  write_only = var.secret
}
