variable "module_name" {
  type = string
}

module "example" {
  source = "./modules/${var.module_name}"
}
