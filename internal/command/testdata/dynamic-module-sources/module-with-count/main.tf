variable "module_name" {
  type  = string
  const = true
}

module "example" {
  source = "./modules/${var.module_name}"
  count  = 2
  number = count.index
}
