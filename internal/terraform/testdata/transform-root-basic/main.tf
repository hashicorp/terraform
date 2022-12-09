terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
    do = {
      source = "digitalocean/digitalocean"
    }
  }
}

provider "aws" {}
resource "aws_instance" "foo" {}

provider "do" {}
resource "do_droplet" "bar" {}
