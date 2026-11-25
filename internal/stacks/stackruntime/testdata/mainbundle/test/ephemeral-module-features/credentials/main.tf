terraform {
  required_providers {
    testing = {
      source = "hashicorp/testing"
    }
  }
}

variable "id" {
  type = string

  validation {
    condition     = startswith(var.id, "credential-")
    error_message = "The data source ID must identify credential keys."
  }
}

data "testing_data_source" "keys" {
  id = provider::testing::echo(var.id)

  lifecycle {
    precondition {
      condition     = var.id != ""
      error_message = "The data source ID must not be empty."
    }
    postcondition {
      condition     = self.value == "a,b"
      error_message = "Expected two credential keys."
    }
  }
}

locals {
  keys = toset(split(",", data.testing_data_source.keys.value))
}

module "credential" {
  source   = "./child"
  for_each = local.keys

  key = each.key
}

output "values" {
  value     = { for key, credential in module.credential : key => credential.value }
  ephemeral = true
}
