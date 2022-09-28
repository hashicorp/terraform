module "child" {
  # NOTE: For this test we need a working absolute path so that Terraform
  # will see this a an "external" module and thus establish a separate
  # package for it, but we won't know which temporary directory this
  # will be in at runtime, so we'll rewrite this file inside the test
  # code to replace %%BASE%% with the actual path. %%BASE%% is not normal
  # Terraform syntax and won't work outside of this test.
  #
  # Note that we're intentionally using the special // delimiter to
  # tell Terraform that it should treat the "package" directory as a
  # whole as a module package, with all of its descendents "downloaded"
  # (copied) together into ./.terraform/modules/child so that child
  # can refer to ../grandchild successfully.
  source = "%%BASE%%/package//child"
}
