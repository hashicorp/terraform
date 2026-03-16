
// this test runs assertions againsts parts of the module that should not
// have executed because of the expected failure. this should be an error
// in the test, but it shouldn't panic or anything like that.

run "fail" {
  variables {
    input = "deny"
  }

  command = plan

  expect_failures = [
    var.input,
  ]

  assert {
    condition = var.followup == "deny"
    error_message = "bad input"
  }

  assert {
    condition = local.input == "deny"
    error_message = "bad local"
  }

  assert {
    condition = module.child.output == "deny"
    error_message = "bad module output"
  }

  assert {
    condition = test_resource.resource.value == "deny"
    error_message = "bad resource value"
  }

  assert {
    condition = output.output == "deny"
    error_message = "bad output"
  }
}
