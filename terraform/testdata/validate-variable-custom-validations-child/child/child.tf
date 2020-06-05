variable "test" {
  type = string

  validation {
    condition     = var.test != "nope"
    error_message = "Value must not be \"nope\"."
  }
}
