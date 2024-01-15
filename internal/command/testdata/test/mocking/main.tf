terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

provider "test" {
  alias = "primary"
}

provider "test" {
  alias = "secondary"
}

variable "instances" {
  type = number
}

variable "child_instances" {
  type = number
}

resource "test_resource" "primary" {
  provider = test.primary
  count = var.instances
}

resource "test_resource" "secondary" {
  provider = test.secondary
  count = var.instances
}

module "child" {
  count = var.instances

  source = "./child"

  providers = {
    test.primary = test.primary
    test.secondary = test.secondary
  }

  instances = var.child_instances
}
