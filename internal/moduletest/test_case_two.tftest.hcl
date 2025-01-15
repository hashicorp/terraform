# test_run_one does a complete apply
run "test_run_one" {
  variables {
    input = "test_run_one"
  }

  assert {
    condition = foo_resource.a.value == "test_run_one"
    error_message = "invalid value"
  }
}

# test_run_two does a refresh only apply
run "test_run_two" {
  plan_options {
    mode = refresh-only
  }

  variables {
    input = "test_run_two"
  }

  assert {
    # value shouldn't change, as we're doing a refresh-only apply.
    condition = foo_resource.a.value == "test_run_one"
    error_message = "invalid value"
  }
}

# test_run_three does an apply with a replace operation
run "test_run_three" {
  variables {
    input = "test_run_three"
  }

  plan_options {
    replace = [
      bar_resource.c
    ]
  }

  assert {
    condition = foo_resource.a.value == "test_run_three"
    error_message = "invalid value"
  }
}
