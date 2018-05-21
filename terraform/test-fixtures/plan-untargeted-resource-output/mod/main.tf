locals {
  one = 1
}

resource "aws_instance" "a" {
  count = "${local.one}"
}

resource "aws_instance" "b" {
  count = "${local.one}"
}

output "output" {
  value = "${join("", coalescelist(aws_instance.a.*.id, aws_instance.b.*.id))}"
}
