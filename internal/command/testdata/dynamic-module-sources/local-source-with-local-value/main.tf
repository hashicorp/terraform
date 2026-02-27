variable "module_name" {
  type  = string
  const = true
}

locals {
  module_path = "./modules/${var.module_name}"
}

module "example" {
  source = local.module_path
}
