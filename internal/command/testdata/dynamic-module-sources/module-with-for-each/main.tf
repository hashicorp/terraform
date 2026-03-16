variable "module_name" {
  type  = string
  const = true
}

module "example" {
  source   = "./modules/${var.module_name}"
  for_each = toset(["a", "b"])
  letter   = each.value
}
