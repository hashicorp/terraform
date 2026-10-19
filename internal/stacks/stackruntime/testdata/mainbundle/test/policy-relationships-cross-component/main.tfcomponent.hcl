required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {
  config {
    ignored = "default"
  }
}

variable "block_deferred" {
  type    = bool
  default = false
}

# The bucket and the block are in different components, so only a run for the
# whole deployment sees both.
component "bucket" {
  source = "./bucket"

  providers = {
    testing = provider.testing.default
  }
}

component "block" {
  source = "./block"

  inputs = {
    deferred = var.block_deferred
  }

  providers = {
    testing = provider.testing.default
  }
}
