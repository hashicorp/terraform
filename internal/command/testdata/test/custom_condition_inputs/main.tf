
variable "input" {
  type = string

  validation {
    condition = var.input == "something very specific"
    error_message = "this should definitely fail"
  }
}

resource "test_resource" "resource" {
  value = var.input
}
