
# This resource gets visited first on the apply walk, but since it DynamicExpands
# to an empty subgraph it ends up being a no-op, leaving the module state
# uninitialized.
resource "test_thing" "a" {
  count = 0
}

# This resource is visited second. During its eval walk we try to build the
# array for the null_resource.a.*.id interpolation, which involves iterating
# over all of the resource in the state. This should succeed even though the
# module state will be nil when evaluating the variable.
resource "test_thing" "b" {
  a_ids = "${join(" ", test_thing.a.*.id)}"
}
