terraform {
  required_version = ">= 0.13.0"
  required_providers {
    tls = {
      source  = "hashicorp/tls"
      version = "~> 2.1.1"
    }
  }
}
