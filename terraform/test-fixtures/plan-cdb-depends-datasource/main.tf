resource "aws_instance" "foo" {
  count = 2
  num     = "2"
  compute = "${element(data.aws_vpc.bar.*.id, count.index)}"
  lifecycle { create_before_destroy = true }
}

data "aws_vpc" "bar" {
  count = 2
  foo = "${count.index}"
}
