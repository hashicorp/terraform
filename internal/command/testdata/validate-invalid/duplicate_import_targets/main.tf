resource "aws_instance" "web" {
}

import {
  to = aws_instance.web
  id = "test"
}

import {
  to = aws_instance.web
  id = "test2"
}
