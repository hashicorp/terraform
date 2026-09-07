variable "child_name" {
  type  = string
  const = true
}

module "parent" {
  source     = "./modules/parent"
  child_name = var.child_name
}
