
module "child_c" {
  # In the unit test where this fixture is used, we treat the source strings
  # as absolute paths rather than as source addresses as we would in a real
  # module walker.
  source = "child_c"
}
