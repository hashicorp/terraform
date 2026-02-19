
run "first" {
  variables {
    input = 2
  }

// var.input2 is ephemeral, and this would cause it to not be set during the plan phase,
// but will be set to default during the apply phase. This leads to the apply update being null
  assert {
    condition = output.output == var.input2
    error_message = "output should have been null"
  }
}
