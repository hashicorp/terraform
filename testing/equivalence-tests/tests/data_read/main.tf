terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.1"
    }
    local = {
      source  = "hashicorp/local"
      version = "2.2.3"
    }
    random = {
      source = "hashicorp/random"
      version = "3.4.3"
    }
  }
}

provider "tfcoremock" {}

provider "local" {}

provider "random" {}

module "create" {
  source = "./create"
  contents = "hello, world!"
}

data "tfcoremock_simple_resource" "read" {
  id = module.create.id

  depends_on = [
    module.create
  ]
}

resource "tfcoremock_simple_resource" "create" {
  string = data.tfcoremock_simple_resource.read.string
}
