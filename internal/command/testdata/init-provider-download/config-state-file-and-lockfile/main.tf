terraform {
  experiments = [pluggable_state_stores]
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "<9.0.0"
    }
  }

  backend "local" {
    path = "./state-using-random-provider.tfstate"
  }
}

resource "random_pet" "maurice" {}
