module "mod" {
  source = "./mod"
  providers = {
    // null may be required by the module, but the name is not defined here
    null = null
  }
}
