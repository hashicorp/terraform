required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "fail_plan" {
  type = bool
  default = false
}

variable "fail_apply" {
  type = bool
  default = false
}

component "self" {
  source = "./"
  providers = {
    testing = provider.testing.default
  }
  inputs = {
    failed_id = "failed"
    resource_id = "resource"
    fail_plan = var.fail_plan
    fail_apply = var.fail_apply
  }
}
