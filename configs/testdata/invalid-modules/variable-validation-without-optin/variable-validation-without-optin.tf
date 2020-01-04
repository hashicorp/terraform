
variable "validation_without_optin" {
  validation { # ERROR: Custom variable validation is experimental
    condition     = var.validation_without_optin != 4
    error_message = "Must not be four."
  }
}
