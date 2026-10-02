run "deprecated_output_plan" {
  command = plan

  assert {
    condition     = output.deprecated_name == "example"
    error_message = "Expected the deprecated output to retain its value."
  }
}

run "deprecated_output_apply" {
  assert {
    condition     = output.deprecated_name == "example"
    error_message = "Expected the deprecated output to retain its value."
  }
}

run "previous_run_output" {
  command = plan

  variables {
    # Passing a deprecated output to a later run preserves its value and produces a warning.
    name = run.deprecated_output_apply.deprecated_name
  }

  assert {
    condition     = output.deprecated_name == "example"
    error_message = "Expected the deprecated output to retain its value."
  }
}
