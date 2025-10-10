terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }

  # Config has been updated to use backend
  # but a state_store block is still represented
  # in the backend state file
  backend "local" {
    path = "local-state.tfstate"
  }
}
