This changelog is ONLY for changes which apply to the 0.7 branch, which is to be rebased on the master branch.

BREAKING CHANGES:

  * core: Plugins now use `hashicorp/go-plugin`. Custom plugins must be recompiled for Terraform 0.7 [GH-5808]

FEATURES:

  * core: State manipulation commands are now exposed via `terraform state`
  * core: Lists can now be specified as variables and outputs [GH-5936]

IMPROVEMENTS:

BUG FIXES:
