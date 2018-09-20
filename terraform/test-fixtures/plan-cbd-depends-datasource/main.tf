resource "aws_instance" "foo" {
  count   = 2
  num     = "2"
  computed = data.aws_vpc.bar[count.index].id

  lifecycle {
    create_before_destroy = true
  }
}

data "aws_vpc" "bar" {
  count = 2
  foo   = count.index
}
