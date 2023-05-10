terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.1"
    }
  }
}

provider "tfcoremock" {}

resource "tfcoremock_multiple_blocks" "multiple_blocks" {
  id = "DA051126-BAD6-4EB2-92E5-F0250DAF0B92"

  first_block {
    id = "D35E88DA-BC3B-46D7-9E0B-4ED4582FA65A"
  }

  first_block {
    id = "E60148A2-04D1-4EF8-90A2-45CAFC02C60D"
  }

  first_block {
    id = "717C64FB-6A93-4763-A1EF-FE4C5B341488"
  }

  second_block {
    id = "157660A9-D590-469E-BE28-83B8526428CA"
  }

  second_block {
    id = "D080F298-2BA4-4DFA-A367-2C5FB0EA7BFE"
  }
}
