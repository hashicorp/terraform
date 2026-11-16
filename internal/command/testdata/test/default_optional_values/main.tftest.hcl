
run "stacked" {
  variables {
    input = {
      required = "required"
      optional = "optional"
      default = "overridden"
    }
  }

  assert {
    condition = output.computed == "overridden"
    error_message = "did not override default value"
  }
}

run "defaults" {
  assert {
    condition = output.computed == "default"
    error_message = "didn't set default value"
  }
}

run "default_matches_last_output" {
  assert {
    condition = var.input == run.defaults.input
    error_message = "output of last should match input of this"
  }
}

run "custom_defined_apply_defaults" {
  variables {
    input = {
      required = "required"
    }
  }

  assert {
    condition = output.computed == "default"
    error_message = "didn't set default value"
  }

  assert {
    condition = var.input == run.defaults.input
    error_message = "output of last should match input of this"
  }
}