variable "module_name" {
  type    = string
  const   = true
  default = "example"
}

module "mod" {
  source = "./modules/${var.module_name}"
}
