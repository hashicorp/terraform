# Set the test-only global "component_instances" to a map to use for the
# for_each expression of the test component.

component "foo" {
  source   = "../modules/with_variable_and_output"
  for_each = _test_only_global.component_instances

  inputs = {
    test = {
      key   = each.key
      value = each.value
    }
  }
}
