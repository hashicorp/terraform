terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.1"
    }
  }
}

provider "tfcoremock" {}

resource "tfcoremock_object" "object" {
  id = "63A9E8E8-71BC-4DAE-A66C-48CE393CCBD3"

  object = {
    string  = "Hello, world!"
    boolean = true
    number  = 10
  }
}
