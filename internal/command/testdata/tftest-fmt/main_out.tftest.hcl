
variables {
  first  = "value"
  second = "value"
}

run "some_run_block" {
  command = plan

  plan_options = {
    refresh = false
  }

  assert {
    condition     = var.input == 12
    error_message = "something"
  }
}
