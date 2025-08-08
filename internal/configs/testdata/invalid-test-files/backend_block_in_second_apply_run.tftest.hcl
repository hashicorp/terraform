run "test_1" {
  command = apply
}

# This run block uses the same internal state as test_1,
# so this the backend block is attempting to load in state
# when there is already non-empty internal state.
run "test_2" {
  command = apply
  backend "local" {
    path = "/tests/other-state"
  }
}
