# This is intended for use as a child stack configuration for situations
# where we're testing the propagation of input variables into the child
# stack.
#
# It contains some input variable declarations that we can assign values to
# for testing purposes. Both are optional to give flexibility for reusing
# this across multiple test cases. If you need something more sophisticated
# for your test, prefer to write a new configuration rather than growing this
# one any further.

variable "test_string" {
  type    = string
  default = null
}

variable "test_map" {
  type    = map(string)
  default = null
}

output "test_string" {
  type  = string
  value = var.test_string
}

output "test_map" {
  type  = map(string)
  value = var.test_map
}
