run "validate_ephemeral_output_plan" {
  command = plan
  variables {
    foo = "whaaat"
  }
  assert {
    condition = output.value == "whaaat"
    error_message = "wrong value"
  }
}
run "validate_ephemeral_output_apply" {
  command = apply
  variables {
    foo = "whaaat"
  }
  assert {
    condition = output.value == "whaaat"
    error_message = "wrong value"
  }
}
