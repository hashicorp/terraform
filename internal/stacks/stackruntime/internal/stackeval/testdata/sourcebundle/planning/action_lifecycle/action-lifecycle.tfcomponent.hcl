# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: BUSL-1.1

required_providers {
  test = {
    source = "terraform.io/builtin/test"
  }
}

provider "test" "main" {
}

component "web" {
  source = "./module_web"

  providers = {
    test = provider.test.main
  }
}

output "result" {
  type  = string
  value = component.web.result
}
