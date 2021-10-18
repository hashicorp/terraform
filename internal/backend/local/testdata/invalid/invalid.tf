# This configuration is intended to be loadable (valid syntax, etc) but to
# fail terraform.Context.Validate.

locals {
  a = local.nonexist
}
