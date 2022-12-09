terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "test_resource" "root" {
  required = module.mod.object.id
}

module "mod" {
  source = "./mod"
}
