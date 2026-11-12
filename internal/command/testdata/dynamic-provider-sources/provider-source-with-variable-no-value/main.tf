variable "provider_src" {
  type  = string
  const = true
}

terraform {
  required_providers {
    test = {
      source = var.provider_src
    }
  }
}
