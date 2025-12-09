terraform {
  required_providers {
    foo = {
      source = "my-org/foo"
    }
  }

  # Config has been updated to use backend
  # but a state_store block is still represented
  # in the backend state file
  backend "local" {
    path = "local-state.tfstate"
  }
}
