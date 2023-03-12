variable "validation" {
  validation {
    condition     = var.validation == 5
    error_message = "Must be five."
  }
}

variable "validation_function" {
  type = list(string)
  validation {
    condition     = length(var.validation_function) > 0
    error_message = "Must not be empty."
  }
}

variable "validation_error_expression" {
  type = list(string)
  validation {
    condition     = length(var.validation_error_expression) < 10
    error_message = "Too long (${length(var.validation_error_expression)} is greater than 10)."
  }
}
