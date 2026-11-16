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

variable "b_deferred" {
  type    = bool
  default = false
}

# Each instance has a bucket and a block with the same value, at the same
# addresses as the other instance's: each walk relates its own, and none
# relates across components.
component "pair" {
  source   = "./pair"
  for_each = toset(["a", "b"])

  inputs = {
    name     = each.key
    deferred = var.b_deferred && each.key == "b"
  }

  providers = {
    testing = provider.testing.default
  }
}
