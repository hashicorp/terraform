terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.1"
    }
  }
}

provider "tfcoremock" {}

resource "tfcoremock_set" "set" {
  id = "046952C9-B832-4106-82C0-C217F7C73E18"
  set = [
    "41471135-E14C-4946-BFA4-2626C7E2A94A",
    "C04762B9-D07B-40FE-A92B-B72AD342658D",
    "D8F7EA80-9E25-4DD7-8D97-797D2080952B",
  ]
}
