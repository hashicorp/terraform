terraform {
  required_providers {
    null = {
      source = "hashicorp/null"
      version = "1.0.0"
    }
  }
}

module "mod" {
  source = "./mod"
  providers = {
    null = null
  }
}
