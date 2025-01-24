resource "aws_instance" "foo" {
  count = 3

  lifecycle {
    concurrency = 1
  }
}

resource "aws_instance" "bar" {
  foo = length(aws_instance.foo)
}
