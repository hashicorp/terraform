
run "first" {
  variables {
    input = 2
  }

  assert {
    condition = output.output != var.input
    error_message = "condition should fail"
  }
}
