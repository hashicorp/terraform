variable "provider_ver" {
  type    = string
  default = "1.0.0"
}

terraform {
  required_providers {
    test = {
      source  = "hashicorp/test"
      version = var.provider_ver
    }
  }
}
