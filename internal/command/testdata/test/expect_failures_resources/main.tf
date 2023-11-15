
variable "input" {
  type = string
}

resource "test_resource" "resource" {
  value = var.input

  lifecycle {
    postcondition {
      condition = self.value != var.input
      error_message = "this really should fail"
    }
  }
}
