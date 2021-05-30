variable "validation" {
  validation {
    condition     = var.validation == 5
    error_message = "Must be five."
  }
}
