
run "first" {
  variables {
    input = 2
  }

  assert {
    condition = output.output == null
    error_message = "output should have been null"
  }
}

run "second" {
  variables {
    input = 8
  }

  assert {
    condition = output.output == 8
    error_message = "output should have been 8"
  }
}
