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
  id = "F40F2AB4-100C-4AE8-BFD0-BF332A158415"

  object = {
    id = "07F887E2-FDFF-4B2E-9BFB-B6AA4A05EDB9"
  }
}
