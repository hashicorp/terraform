required_providers {
  testing = {
    // The source is wrong, so validate should complain.
    source  = "hashicorp/wrong"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "input" {
  type = string
}

component "self" {
  source = "../"

  providers = {
    // Everything looks okay here, but the provider types are actually wrong.
    testing = provider.testing.default
  }

  inputs = {
    input = var.input
  }
}
