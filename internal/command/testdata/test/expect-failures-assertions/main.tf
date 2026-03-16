
variable "input" {
  type = string

  validation {
    condition = var.input == "allow"
    error_message = "invalid input value"
  }
}

variable "followup" {
  type = string
  default = "allow"

  validation {
    condition = var.followup == var.input
    error_message = "followup must match input"
  }
}

locals {
  input = var.followup
}

module "child" {
  source = "./child"
  input = var.input
}

resource "test_resource" "resource" {
  value = local.input
}

output "output" {
  value = var.input
}
