terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.1"
    }
  }
}

provider "tfcoremock" {}

resource "tfcoremock_simple_resource" "json" {
  string = "{\"list-attribute\":[\"one\",\"four\",\"three\"],\"object-attribute\":{\"key_one\":\"value_one\",\"key_three\":\"value_two\", \"key_four\":\"value_three\"},\"string-attribute\":\"a new string\"}"
}
