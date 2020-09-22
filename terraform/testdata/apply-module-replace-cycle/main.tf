module "a" {
  source = "./mod1"
}

module "b" {
  source = "./mod2"
  ids = module.a.ids
}
