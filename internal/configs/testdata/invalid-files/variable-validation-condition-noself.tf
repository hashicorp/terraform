locals {
  something = "else"
}

variable "validation" {
  validation {
    condition     = local.something == "else"
    error_message = "Something else."
  }
}
