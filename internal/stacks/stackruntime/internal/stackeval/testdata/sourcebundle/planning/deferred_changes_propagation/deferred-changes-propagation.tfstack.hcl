
required_providers {
  test = {
    source = "terraform.io/builtin/test"
  }
}

provider "test" "main" {
}

variable "first_count" {
  type = number
}

component "first" {
  source = "./"

  inputs = {
    instance_count = var.first_count
  }
  providers = {
    test = provider.test.main
  }
}

component "second" {
  source = "./"

  inputs = {
    instance_count = component.first.constant_one
  }
  providers = {
    test = provider.test.main
  }
}
