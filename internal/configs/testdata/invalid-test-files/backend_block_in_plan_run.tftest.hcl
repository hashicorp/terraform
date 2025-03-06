# This backend block is used in a plan run block
# They're expected to be used in the first apply run block
# for a given state key
run "setup" {
  command = plan
  backend "local" {
    path = "/tests/other-state"
  }
}

run "test" {
  command = apply
}
