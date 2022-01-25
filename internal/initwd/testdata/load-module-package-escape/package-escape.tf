module "child" {
  # NOTE: For this test we need a working absolute path so that Terraform
  # will see this a an "external" module and thus establish a separate
  # package for it, but we won't know which temporary directory this
  # will be in at runtime, so we'll rewrite this file inside the test
  # code to replace %%BASE%% with the actual path. %%BASE%% is not normal
  # Terraform syntax and won't work outside of this test.
  source = "%%BASE%%/child"
}
