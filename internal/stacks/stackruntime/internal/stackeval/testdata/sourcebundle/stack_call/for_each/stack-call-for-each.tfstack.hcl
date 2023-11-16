# Set the test-only global "child_stack_for_each" to a map conforming
# to the following type constraint:
#
# map(object({
#   test_string = optional(string)
#   test_map    = optional(map(string))
# }))

stack "child" {
  source   = "../with_variables_and_outputs"
  for_each = _test_only_global.child_stack_for_each

  inputs = each.value
}
