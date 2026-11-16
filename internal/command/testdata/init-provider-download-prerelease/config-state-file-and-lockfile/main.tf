terraform {
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "1.2.3-beta"
    }
  }

  backend "local" {
    path = "./state-using-random-provider.tfstate"
  }
}

resource "random_pet" "maurice" {}
