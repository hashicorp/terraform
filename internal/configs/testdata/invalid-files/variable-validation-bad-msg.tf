variable "validation" {
  validation {
    condition     = var.validation != 4
    error_message = "" # ERROR: Invalid validation error message
  }
}
