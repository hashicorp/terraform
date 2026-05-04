terraform {
  required_providers {
    test = {
      source  = "hashicorp/test"
      version = "1.2.3"
    }
  }
}

provider "test" {}

module "child" {
  source = "./modules/child"
}
