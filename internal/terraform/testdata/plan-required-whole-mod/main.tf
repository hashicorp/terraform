resource "test_resource" "root" {
  required = local.object.id
}

locals {
  # This indirection is here to force the evaluator to produce the whole
  # module object here rather than just fetching the single "object" output.
  # This makes this fixture different than plan-required-output, which just
  # accesses module.mod.object.id directly and thus visits a different
  # codepath in the evaluator.
  mod    = module.mod
  object = local.mod.object
}

module "mod" {
  source = "./mod"
}
