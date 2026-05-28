variable "provider_src" {
  type    = string
  default = "nonconst"
}

terraform {
  required_providers {
    test = {
      source = var.provider_src
    }
  }
}
