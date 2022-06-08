
locals {
  foo = 1
}

variable "validation" {
  validation {
    condition     = local.foo == var.validation # ERROR: Invalid reference in variable validation
    error_message = "Must be five."
  }
}

variable "validation_error_expression" {
  validation {
    condition     = var.validation_error_expression != 1
    error_message = "Cannot equal ${local.foo}." # ERROR: Invalid reference in variable validation
  }
}
