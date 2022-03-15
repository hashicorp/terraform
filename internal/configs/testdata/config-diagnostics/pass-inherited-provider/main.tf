provider "test" {
  value = "ok"
}

module "mod" {
  source = "./mod"
}

# FIXME: This test is for an awkward interaction that we've preserved for
# compatibility with what was arguably a bug in earlier versions: if a
# child module tries to use an inherited provider configuration explicitly by
# name then Terraform would historically use the wrong provider configuration.
#
# Since we weren't able to address that bug without breaking backward
# compatibility, instead we emit a warning to prompt the author to be explicit,
# passing in the configuration they intend to use.
#
# This case is particularly awkward because a change in the child module
# (previously referring to a provider only implicitly, but now naming it
# explicitly) can cause a required change in _this_ module (the caller),
# even though the author of the child module would've seen no explicit warning
# that they were making a breaking change. Hopefully we can improve on this
# in a future language edition.
