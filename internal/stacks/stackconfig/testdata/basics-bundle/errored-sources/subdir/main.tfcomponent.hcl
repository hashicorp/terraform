required_providers {
  null = {
    source  = "hashicorp/null"
    version = "3.2.1"
  }
}

provider "null" "a" {}

component "a" {
  source = "../"

  inputs = {
    name = var.name
  }

  providers = {
    null = provider.null.a
  }
}
