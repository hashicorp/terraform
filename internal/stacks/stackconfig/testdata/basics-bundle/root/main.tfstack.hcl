stack "nested" {
  source  = "example.com/awesomecorp/nested/happycloud"
  version = "< 2.0.0"

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

# component "a" {
#   source = "../"

#   inputs = {
#     name = var.name
#   }
#   providers = {
#     null = var.provider
#   }
# }

removed {
  from = component.a

  source = "../"

  providers = {
    null = var.provider
  }

  # lifecycle {
  #   destroy = false
  # }
}
