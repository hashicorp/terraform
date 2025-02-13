# There cannot be two backend blocks in a single run block
run "setup" {
  backend "local" {
    path = "/tests/state"
  }
  backend "local" {
    path = "/tests/other-state"
  }
}

run "test" {
}
