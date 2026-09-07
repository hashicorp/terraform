
run "test" {
  command = plan

  plan_options {
    target = [test_resource.two]
  }

  variables {
    input = "hello"
  }

  assert {
    condition = var.input == "hello"
    error_message = "wrong input"
  }
}