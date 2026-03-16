run "setup" {
  command = apply

  backend "local" {
    path = "/tests/state/terraform.tfstate"
  }
}

# "test" uses the same internal state file as "setup", which has already loaded state from a backend block
# and is an apply run block.
# The backend block can only occur once in a given set of run blocks that share state.
run "test" {
  command = apply

  backend "local" {
    path = "/tests/state/terraform.tfstate"
  }
}
