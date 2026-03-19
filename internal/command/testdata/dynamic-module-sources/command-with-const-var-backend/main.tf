terraform {
  backend "local-vars" {

  }
}

variable "module_name" {
  type  = string
  const = true
}

module "child" {
  source = "./modules/${var.module_name}"
}
