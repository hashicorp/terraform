# The "state/terraform.tfstate" local backend is used with the user-supplied internal state "foobar-1"
run "test_1" {
  state_key = "foobar-1"
  backend "local" {
    path = "state/terraform.tfstate"
  }
}

# The "state/terraform.tfstate" local backend is used with the user-supplied internal state "foobar-2"
run "test_2" {
  state_key = "foobar-2"
  backend "local" {
    path = "state/terraform.tfstate"
  }
}
