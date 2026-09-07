variable "module_name" {
  type  = string
  const = true
}

locals {
  module_source = "./modules/${var.module_name}"
}

module "mod" {
  source = local.module_source
}
