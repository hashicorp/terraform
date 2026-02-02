# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: BUSL-1.1

required_providers {
  testing = {
    source = "terraform.io/builtin/testing"
  }
}

provider "testing" "main" {
}

component "web" {
  source = "./module_web"

  providers = {
    testing = provider.testing.main
  }
}

output "result" {
  type  = string
  value = component.web.result
}
