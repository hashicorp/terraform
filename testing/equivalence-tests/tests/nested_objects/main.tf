terraform {
  required_providers {
    tfcoremock = {
      source = "hashicorp/tfcoremock"
      version = "0.1.1"
    }
  }
}

provider "tfcoremock" {}

resource "tfcoremock_nested_object" "nested_object" {
  id = "B2491EF0-9361-40FD-B25A-0332A1A5E052"

  parent_object = {
    first_nested_object = {
      attribute_one = "09AE7244-7BFB-476B-912C-D1AB4E7E9622",
      attribute_two = "5425587C-49EF-4C1E-A906-1DC923A12725"
    }
    second_nested_object = {
      attribute_one = "63712BFE-78F8-42D3-A074-A78249E5E25E",
      attribute_two = "FB350D92-4AAE-48C6-A408-BFFAFAD46B04"
    }
  }
}
