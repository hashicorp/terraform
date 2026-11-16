
terraform {
  required_providers {
    test = {
      source = "terraform.io/builtin/test"
    }
  }
}

variable "instance_count" {
  type = number
}

resource "test" "a" {
  # This one has on intrinsic need to be deferred, but
  # should still be deferred when an upstream component
  # has a deferral.
}

resource "test" "b" {
  count = var.instance_count
}

output "constant_one" {
  value = 1
}
