terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
      configuration_aliases = [test.primary, test.secondary]
    }
  }
}

variable "instances" {
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

output "primary" {
  value = test_resource.primary
}

output "secondary" {
  value = test_resource.secondary
}
