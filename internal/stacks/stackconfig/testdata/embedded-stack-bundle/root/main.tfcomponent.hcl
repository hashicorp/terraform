required_providers {
  null = {
    source  = "hashicorp/null"
    version = "3.2.1"
  }
}


stack "pet-nulls" {
  source  = "example.com/awesomecorp/tfstack-pet-nulls"
  version = "0.0.2"

  inputs = {
    name     = var.name
    provider = provider.null.a
  }
}

provider "null" "a" {
}

locals {
  sound = "bleep bloop"
}
