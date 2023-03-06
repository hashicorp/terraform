terraform {
  required_providers {
    tfcoremock = {
      source = "hashicorp/tfcoremock"
      version = "0.1.1"
    }
  }
}

provider "tfcoremock" {}

resource "tfcoremock_nested_map" "nested_map" {
  id = "502B0348-B796-4F6A-8694-A5A397237B85"

  maps = {
    "first_nested_map": {
      "first_key": "6E80C701-A823-43FE-A520-699851EF9052",
      "second_key": "D55D0E1E-51D9-4BCE-9021-7D201906D3C0"
      "third_key": "79CBEBB1-1192-480A-B4A8-E816A1A9D2FC"
    },
    "second_nested_map": {
      "first_key": "9E858021-953F-4DD3-8842-F2C782780422",
    }
  }
}
