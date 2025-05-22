
variable "v" {
  description = "in root module"
  default     = ""
}

module "child" {
  source = "./modules/child_a"
}
