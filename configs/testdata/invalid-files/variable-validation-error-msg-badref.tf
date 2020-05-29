
locals {
  foo = 1
}

variable "validation" {
  default = 1
  validation {
    condition     = var.validation == 1
    error_message = "Must be ${local.foo}." # ERROR: Invalid reference in variable validation
  }
}
