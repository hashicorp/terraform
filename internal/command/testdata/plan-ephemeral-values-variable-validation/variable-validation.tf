variable "test" {
  ephemeral = true
  default   = "foo"

  validation {
    condition     = var.test != "foo"
    error_message = "value"
  }
}
