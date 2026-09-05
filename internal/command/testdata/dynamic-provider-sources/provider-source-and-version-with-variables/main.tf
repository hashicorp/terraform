variable "provider_src" {
  type  = string
  const = true
}

variable "provider_ver" {
  type  = string
  const = true
}

terraform {
  required_providers {
    test = {
      source  = var.provider_src
      version = var.provider_ver
    }
  }
}
