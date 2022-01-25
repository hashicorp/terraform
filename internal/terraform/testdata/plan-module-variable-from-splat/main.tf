module "mod1" {
  source = "./mod"
  param  = ["this", "one", "works"]
}

module "mod2" {
  source = "./mod"
  param  = [module.mod1.out_from_splat[0]]
}
