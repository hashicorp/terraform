variable "provider_src" {
  type    = string
  const   = true
  default = "hashicorp/test"
}

terraform {
  required_providers {
    test = {
      source = var.provider_src
    }
  }
}
