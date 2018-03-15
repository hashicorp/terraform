
variable "v" {
  description = "in child_a module"
  default     = ""
}

module "child_b" {
  source = "./child_b"
}
