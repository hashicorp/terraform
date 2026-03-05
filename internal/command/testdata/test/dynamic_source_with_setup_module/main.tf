variable "module_name" {
  type    = string
  const   = true
  default = "example"
}

variable "managed_id" {
  type = string
}

module "mod" {
  source = "./modules/${var.module_name}"
  id     = var.managed_id
}
