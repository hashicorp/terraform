variable "validation" {
  validation {
    condition     = var.validation != 4
    error_message = "not four" # ERROR: Invalid validation error message
  }
}
