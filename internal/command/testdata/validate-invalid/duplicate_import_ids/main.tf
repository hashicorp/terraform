resource "aws_instance" "web" {
}

resource "aws_instance" "other_web" {
}

import {
  to = aws_instance.web
  id = "test"
}

import {
  to = aws_instance.other_web
  id = "test"
}
