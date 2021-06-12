variable "validation" {
  validation {
    condition     = var.validation != 4
    error_message = "not four" # WARNING: Validation error message should use consistent writing style
  }
}
