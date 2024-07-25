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

data "aws_secretsmanager_secret" "creds" {}

module "child" {
  source = "./child"
}
