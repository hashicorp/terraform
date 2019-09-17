variable "input" {}

provider "aws" {
  region = "us-east-${var.input}"
}

resource "aws_instance" "foo" { }
