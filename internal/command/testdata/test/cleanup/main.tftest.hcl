run "test" {
  variables {
    id = "test"
  }
}

run "test_two" {
  skip_cleanup = true # This will leave behind the state
  variables {
    id = "test_two"
  }
}

run "test_three" {
  state_key = "state_three"
  variables {
    id = "test_three"
    destroy_fail = true // This will fail to destroy and leave behind the state
  }
}

run "test_four" {
  variables {
    id = "test_four"
  }
}