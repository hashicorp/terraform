provider "test" {
  value = "ok"
}

module "mod" {
  source = "./mod"
}
