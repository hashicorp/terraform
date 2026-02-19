
run "first" {
  variables {
    interesting_input = "bar"
  }

  assert {
    condition     = test_resource.resource.value == output.null_output
    error_message = "this is always going to fail"
  }

  assert {
    condition     = var.null_input == output.null_output
    error_message = "this should pass"
  }
}
