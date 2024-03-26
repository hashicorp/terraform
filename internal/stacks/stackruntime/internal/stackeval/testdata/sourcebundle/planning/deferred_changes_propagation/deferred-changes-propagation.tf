
terraform {
  required_providers {
    test = {
      source = "terraform.io/builtin/test"
    }
  }

  # TODO: Remove this if this experiment gets stabilized.
  # If you're removing this, remember to also update the calling test so
  # that it no longer enables the use of experiments, to ensure that we're
  # really not depending on any experimental features.
  experiments = [unknown_instances]
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
