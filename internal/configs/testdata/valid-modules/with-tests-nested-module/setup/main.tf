
variable "value" {
  type = string
}

module "child" {
  source = "./other"

  value = var.value
}

output "value" {
  value = module.child.value
}
