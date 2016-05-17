variable "mod_count_root" {
  type = "string"
  default = "3"
}

module "child" {
  source    = "./child"
  mod_count_child = "${var.mod_count_root}"
}
