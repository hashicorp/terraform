variable "module_src" {
  type    = string
  default = "./modules/simple"
  const   = true
}

module "const_var_source" {
  source = var.module_src
}
