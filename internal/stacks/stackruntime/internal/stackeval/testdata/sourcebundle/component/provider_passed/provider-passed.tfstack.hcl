required_providers {
  tfcoremock = {
    source  = "hashicorp/tfcoremock"
    version = "0.2.0"
  }
}

provider "tfcoremock" "this" {}

component "foo" {
  source = "../modules/with_provider"

  inputs = {}

  providers = {
    tfcoremock = provider.tfcoremock.this
  }
}
