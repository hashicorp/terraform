required_providers {
  other = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "other" "default" {}

variable "input" {
  type = string
}

component "self" {
  source = "../"

  providers = {
    // Even though the names are wrong, the underlying types are the same
    // so this should be okay.
    testing = provider.other.default
  }

  inputs = {
    input = var.input
  }
}
