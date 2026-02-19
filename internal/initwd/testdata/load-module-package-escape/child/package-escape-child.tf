module "grandchild" {
  # NOTE: This seems like it ought to work because there is indeed a
  # ../grandchild directory, but our caller loaded us as an external
  # module using an absolute path and so we're actually isolated from
  # the parent directory in a separate "module package", and so we
  # can't traverse out to find the grandchild module.
  source = "../grandchild"
}
