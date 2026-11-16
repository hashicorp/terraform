variable "test" {
  sensitive = true
  default = "nope"
}

module "child" {
  source = "./child"

  test = var.test
}
