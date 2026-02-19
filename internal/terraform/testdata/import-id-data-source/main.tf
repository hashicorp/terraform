data "aws_subnet" "bar" {
  vpc_id     = "abc"
  cidr_block = "10.0.1.0/24"
}

import {
  to = aws_subnet.bar
  id = data.aws_subnet.bar.id
}

resource "aws_subnet" "bar" {}
