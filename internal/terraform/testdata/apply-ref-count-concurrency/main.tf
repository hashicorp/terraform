lock "aws" {
  concurrency = 1
}

resource "aws_instance" "foo" {
  count = 3

  lifecycle {
    lock = lock.aws
  }
}

resource "aws_instance" "bar" {
  foo = length(aws_instance.foo)
}

data "aws_data_source" "baz" {
  count = 2

  lifecycle {
    lock = lock.aws
  }
}
