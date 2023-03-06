terraform {
  required_providers {
    do = {
      source = "hashicorp/do"
    }
  }
}

resource "do_instance" "foo" {
    num = "2"
}

resource "aws_instance" "bar" {
    foo = "bar"
}
