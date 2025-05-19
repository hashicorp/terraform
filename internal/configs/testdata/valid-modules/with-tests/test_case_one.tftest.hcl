variables {
  input = "default"
}

# test_run_one runs a partial plan
run "test_run_one" {
  command = plan

  plan_options {
    target = [
      foo_resource.a
    ]
  }

  assert {
    condition = foo_resource.a.value == "default"
    error_message = "invalid value"
  }
}

# test_run_two does a complete apply operation
run "test_run_two" {
  variables {
    input = "custom"
  }

  assert {
    condition = foo_resource.a.value == "custom"
    error_message = "invalid value"
  }
}
