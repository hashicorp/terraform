terraform {
  experiments = [variable_validation] # WARNING: Experimental feature "variable_validation" is active
}

variable "validation" {
  validation {
    condition     = var.validation == 5
    error_message = "Must be five."
  }
}
