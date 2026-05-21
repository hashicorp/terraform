terraform {
  required_providers {
    test = {
      source = "test"
    }
  }
}

variable "module_input" {
  type = string
  validation {
    condition     = provider::test::is_true(var.module_input == "hello")
    error_message = "The module_input variable must be set to \"hello\""
  }
}

module "example" {
  source = "./modules/example"
  in     = var.module_input
}
