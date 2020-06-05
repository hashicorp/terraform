provider "aws" {
  foo = "bar"
}

provider "aws_elb" {
  foo = "baz"
}

resource "aws_instance" "foo" {}
resource "aws_instance" "bar" {}
resource "aws_elb" "lb" {}
resource "do_droplet" "bar" {}
