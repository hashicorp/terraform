variable "test_var" {
    default = "bar"
}
resource "test_instance" "test" {
    ami = var.test_var
}

output "test" {
    value = var.test_var
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
