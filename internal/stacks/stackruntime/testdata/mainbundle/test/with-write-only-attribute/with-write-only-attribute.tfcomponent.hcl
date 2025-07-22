required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "providers" {
  type = set(string)
}

provider "testing" "main" {
  for_each = var.providers
}

component "main" {
  source = "./"

  providers = {
    testing = provider.testing.main["single"]
  }

  inputs = {
    datasource_id = "datasource"
    resource_id = "resource"
    write_only_input = "secret"
  }
}
