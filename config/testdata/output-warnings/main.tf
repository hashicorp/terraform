locals {
  "count" = 1
}

resource "test_instance" "foo" {
  count = "${local.count}"
}

output "foo_id" {
  value = "${test_instance.foo.id}"
}

variable "condition" {
  default = "true"
}

resource "test_instance" "bar" {
  count = "${var.condition ? 1 : 0}"
}


output "bar_id" {
  value = "${test_instance.bar.id}"
}
