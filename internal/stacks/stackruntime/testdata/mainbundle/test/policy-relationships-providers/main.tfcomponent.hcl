required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "a" {
  config {
    ignored = "x"
  }
}

# The same configuration as a, so the same class, but another instance.
provider "testing" "same_as_a" {
  config {
    ignored = "x"
  }
}

provider "testing" "keyed" {
  for_each = toset(["y"])

  config {
    ignored = each.key
  }
}

# a1 and a2 share provider.testing.a: one provider instance in the run.
component "a1" {
  source = "./bucket"

  providers = {
    testing = provider.testing.a
  }
}

component "a2" {
  source = "./bucket"

  providers = {
    testing = provider.testing.a
  }
}

component "same_as_a" {
  source = "./bucket"

  providers = {
    testing = provider.testing.same_as_a
  }
}

component "keyed" {
  source = "./bucket"

  providers = {
    testing = provider.testing.keyed["y"]
  }
}
