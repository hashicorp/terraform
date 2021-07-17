variable "validation" {
  validation {
    condition = var.validation != 4
    # ERROR: Missing required argument
  }
}
