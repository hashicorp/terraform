module "mod" {
  source = "./mod"
  providers = {
    // bar may be required by the module, but the name is not defined here
    bar = bar
  }
}
