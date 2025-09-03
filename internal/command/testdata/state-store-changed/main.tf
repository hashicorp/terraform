terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {
      region = "changed-value" # changed versus backend state file
    }

    value = "changed-value" # changed versus backend state file
  }
}
