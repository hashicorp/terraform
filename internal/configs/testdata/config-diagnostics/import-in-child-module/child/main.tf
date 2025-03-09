resource "aws_instance" "web" {}

import {
  to = aws_instance.web
  id = "test"
}