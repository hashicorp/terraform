variable "instance_count" {}

resource "aws_instance" "one" {
  count = var.instance_count
}

locals {
  one_id = element(concat(aws_instance.one.*.id, list("")), 0)
}

resource "aws_instance" "two" {
  val = local.one_id
}
