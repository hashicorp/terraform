variable "module_name" {
  type    = string
  default = "simple"
  const   = true
}

locals {
  module_src = "../modules/${var.module_name}"
}

module "const_var_source" {
  source = local.module_src
}
