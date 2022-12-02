terraform {
  experiments = [smoke_tests]
}

variable "a" {
  type    = string
}

variable "b" {
  type    = string
}

smoke_test "try" {
  precondition {
    condition     = var.a == "a"
    error_message = "A isn't."
  }

  postcondition {
    condition     = var.b == "b"
    error_message = "B isn't."
  }
}
