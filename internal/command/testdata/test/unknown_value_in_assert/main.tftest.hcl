
run "one" {
  command = plan
}

run "two" {
  assert {
    condition = output.destroy_fail == run.one.destroy_fail
    error_message = "should fail"
  }
}
