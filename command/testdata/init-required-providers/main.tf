terraform {
  required_providers {
    // legacy syntax
    test = "1.2.3"

    // 0.13 syntax
    source = {
      version = "1.2.3"
    }

    default = {
      source = "registry.terraform.io/hashicorp/random"
    }
  }
}
