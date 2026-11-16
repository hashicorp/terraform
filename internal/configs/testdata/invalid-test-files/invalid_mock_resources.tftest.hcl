mock_provider "aws" {

  mock_resource "aws_instance" {}

  mock_resource "aws_ami_instance" {
    defaults = {
      ami = var.ami
    }
  }

}

run "test" {}
