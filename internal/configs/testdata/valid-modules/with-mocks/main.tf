terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}

resource "aws_instance" "first" {}

resource "aws_instance" "second" {}

resource "aws_instance" "third" {}

resource "aws_instance" "fourth" {
  provisioner "local-exec" {
    command = ""
  }
}

data "aws_secretsmanager_secret" "creds" {}

module "child" {
  source = "./child"
}
