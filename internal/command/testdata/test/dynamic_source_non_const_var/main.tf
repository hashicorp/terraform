variable "module_name" {
  type = string
}

module "mod" {
  source = "./modules/${var.module_name}"
}
