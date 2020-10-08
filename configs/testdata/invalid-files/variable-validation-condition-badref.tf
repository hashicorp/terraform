
locals {
  foo = 1
}

variable "validation" {
  validation {
    condition     = local.foo == var.validation # ERROR: Invalid reference in variable validation
    error_message = "Must be five."
  }
}
