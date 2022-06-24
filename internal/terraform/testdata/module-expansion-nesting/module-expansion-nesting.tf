
# This configuration includes a bunch of module nesting in support of
# benchmarking ModuleExpansionTransformer in BenchmarkModuleExpansionTransformer.
#
# The transformer recursively visits all modules in the tree, and so the shape
# of this tree is intended to allow the benchmark to be sensitive to
# differences between performance costs that:
# - scale by total number of distinct modules, regardless of tree shape
# - scale by the depth of nesting of the module tree
# - are fixed, regardless of number of modules or tree shape

module "a" {
  count = 1

  source = "./a"
}

module "b" {
  count = 1

  source = "./b"
}

resource "foo" "bar" {}
