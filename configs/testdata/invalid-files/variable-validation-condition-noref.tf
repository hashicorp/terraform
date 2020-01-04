
terraform {
  experiments = [variable_validation]
}

variable "validation" {
  validation {
    condition     = true # ERROR: Invalid variable validation condition
    error_message = "Must be true."
  }
}
