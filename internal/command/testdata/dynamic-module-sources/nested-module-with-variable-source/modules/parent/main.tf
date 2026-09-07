variable "child_name" {
  type  = string
  const = true
}

module "child" {
  source = "../${var.child_name}"
}
