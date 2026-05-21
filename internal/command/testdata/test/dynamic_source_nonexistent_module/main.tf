variable "module_name" {
  type    = string
  const   = true
  default = "nonexistent"
}

module "mod" {
  source = "./modules/${var.module_name}"
}
