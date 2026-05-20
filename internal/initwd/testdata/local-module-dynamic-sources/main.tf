module "name" {
  source = "./${var.path}"
}

variable "path" {
  type    = string
  default = "child"
}
