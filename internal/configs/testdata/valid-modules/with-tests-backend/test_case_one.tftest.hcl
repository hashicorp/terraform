variables {
  input = "default"
}

run "load_state" {
  backend "local" {
    path = "state/terraform.tfstate"
  }
}

run "test_run" {
  variables {
    input = "custom"
  }

  assert {
    condition     = foo_resource.a.value == "custom"
    error_message = "invalid value"
  }
}
