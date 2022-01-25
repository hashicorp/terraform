terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
      version = ">= 1.2.3"
    }
  }
}

provider "test" {
  region = "somewhere"
  version = "1.2.3"
}

variable "test_var" {
  default = "bar"
}

resource "test_instance" "test" {
  ami   = var.test_var
  count = 3
}

output "test" {
  value = var.test_var
}
