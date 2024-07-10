# Removal of this component
# component "self" {
#   source = "./"
#   inputs = {
#   }
#   providers = {
#     testing = provider.testing.default
#   }
# }

required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

removed {
    from = component.self
    source = "./"

    providers = {
      testing = provider.testing.default
    }
}
