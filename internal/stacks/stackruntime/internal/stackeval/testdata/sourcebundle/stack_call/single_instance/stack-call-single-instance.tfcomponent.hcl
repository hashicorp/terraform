# Set the test-only global "child_stack_inputs" to an object conforming
# to the following type constraint:
#
# object({
#   test_string = optional(string)
#   test_map    = optional(map(string))
# })

stack "child" {
  source = "../with_variables_and_outputs"

  inputs = _test_only_global.child_stack_inputs
}
