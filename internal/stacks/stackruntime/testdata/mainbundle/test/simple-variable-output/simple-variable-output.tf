terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "input_value" {
  type = string
}

# Output that depends only on input variable - should be available during pre-apply
output "simple_result" {
  value = "processed-${var.input_value}"
}

# Output that computes a derived value - should also be available during pre-apply
output "computed_result" {
  value = "${var.input_value}-computed"
}