
module "child_c" {
  # Note: this test case has an unrealistic module loader that resolves all
  # sources as relative to the fixture directory, rather than to the
  # current module directory as Terraform normally would.
  source = "./child_c"
}
