# There cannot be two backend blocks in a single run block
run "setup" {
  backend "local" {
    path = "/tests/state/terraform.tfstate"
  }
  backend "local" {
    path = "/tests/other-state/terraform.tfstate"
  }
}

run "test" {
}
