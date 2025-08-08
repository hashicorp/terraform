variables {
  input = "default"
}

# The backend in "load_state" is used to set an internal state without an explicit key
run "load_state" {
  backend "local" {
    path = "state/terraform.tfstate"
  }
}

# "test_run" uses the same internal state as "load_state"
run "test_run" {
  variables {
    input = "custom"
  }

  assert {
    condition     = foo_resource.a.value == "custom"
    error_message = "invalid value"
  }
}
