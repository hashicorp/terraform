variable "provider_src" {
  type  = string
  const = true
}

locals {
  provider_source = var.provider_src
}

terraform {
  required_providers {
    test = {
      source = local.provider_source
    }
  }
}
