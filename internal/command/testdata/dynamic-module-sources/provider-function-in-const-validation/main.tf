terraform {
  required_providers {
    test = {
      source = "test"
    }
  }
}

variable "module_name" {
  type  = string
  const = true
  validation {
    condition     = provider::test::is_true(var.module_name == "example")
    error_message = "The module_name variable must be set to \"example\""
  }
}

module "example" {
  source = "./modules/${var.module_name}"
}
