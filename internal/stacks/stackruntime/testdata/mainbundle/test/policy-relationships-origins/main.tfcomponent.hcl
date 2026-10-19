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

# Both buckets of a component have the id "shared", so only an origin tells
# which one a block refers to.
component "bucket" {
  source = "./buckets"

  providers = {
    testing = provider.testing.default
  }
}

component "keyed_bucket" {
  source   = "./buckets"
  for_each = toset(["a"])

  providers = {
    testing = provider.testing.default
  }
}

# A direct reference to another component's output: the block's value has
# the origin of the output, testing_resource.bad's id.
component "direct" {
  source = "./block"

  inputs = {
    bucket_id = component.bucket.bad_id
  }

  providers = {
    testing = provider.testing.default
  }
}

# Not a direct reference: the block's value has no origin (VARIABLE) and
# relates by value.
component "computed" {
  source = "./block"

  inputs = {
    bucket_id = lower(component.bucket.bad_id)
  }

  providers = {
    testing = provider.testing.default
  }
}

# A direct reference to the instance of another component with the same key.
component "keyed" {
  source   = "./block"
  for_each = toset(["a"])

  inputs = {
    bucket_id = component.keyed_bucket[each.key].good_id
  }

  providers = {
    testing = provider.testing.default
  }
}

# Through another component's input and output: the origin is still the
# bucket's.
component "relay" {
  source = "./relay"

  inputs = {
    in = component.bucket.good_id
  }
}

component "chained" {
  source = "./block"

  inputs = {
    bucket_id = component.relay.out
  }

  providers = {
    testing = provider.testing.default
  }
}
