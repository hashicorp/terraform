# Set the test-only global "component_inputs" to an object.
#
# The child module we're using here expects a single input value of any type
# called "test", and will echo it back verbatim as an output value also called
# "test".

component "foo" {
  source = "../modules/with_variable_and_output"

  inputs = _test_only_global.component_inputs
}
