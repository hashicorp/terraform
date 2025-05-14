terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

removed {
  from = testing_resource.resource

  lifecycle {
    destroy = false
  }
}
