
run "one" {
  command = plan
}

run "two" {
  variables {
    destroy_fail = run.one.destroy_fail
  }
}
