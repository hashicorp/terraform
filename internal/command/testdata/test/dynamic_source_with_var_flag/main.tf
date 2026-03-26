variable "module_name" {
  type  = string
  const = true
}

module "mod" {
  source = "./modules/${var.module_name}"
}
