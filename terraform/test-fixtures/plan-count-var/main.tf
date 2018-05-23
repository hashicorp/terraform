variable "instance_count" {}

resource "aws_instance" "foo" {
  count = var.instance_count
  foo   = "foo"
}

resource "aws_instance" "bar" {
  foo = join(",", aws_instance.foo.*.foo)
}
