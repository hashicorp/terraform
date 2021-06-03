terraform {
  required_providers {
    boop = {
      source = "foobar/beep" # intentional mismatch between local name and type
    }
  }
}

resource "aws_instance" "no_count" {
}

resource "aws_instance" "count" {
  count = 1
}

resource "boop_instance" "yep" {
}

resource "boop_whatever" "nope" {
}

data "beep" "boop" {
}
