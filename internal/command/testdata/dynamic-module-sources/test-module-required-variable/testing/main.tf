variable "module_src" {
  type  = string
  const = true
}

module "const_var_source" {
  source = var.module_src
}
