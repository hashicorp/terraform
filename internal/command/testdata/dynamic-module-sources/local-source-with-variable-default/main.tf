variable "module_name" {
  type    = string
  const   = true
  default = "example"
}

module "example" {
  source = "./modules/${var.module_name}"
}
