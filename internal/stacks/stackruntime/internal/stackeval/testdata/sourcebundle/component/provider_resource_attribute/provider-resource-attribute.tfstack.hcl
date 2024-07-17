required_providers {
  tfcoremock = {
    source  = "hashicorp/tfcoremock"
    version = "0.2.0"
  }
}

provider "tfcoremock" "this" {}

component "foo" {
  source = "../modules/with_provider_resource_attribute"

  inputs = {}

  providers = {
    tfcoremock = provider.tfcoremock.this
  }
}
