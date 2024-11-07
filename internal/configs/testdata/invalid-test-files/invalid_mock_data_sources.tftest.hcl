mock_provider "aws" {

  mock_data "aws_instance" {}

  mock_data "aws_ami_instance" {
    defaults = {
      ami = var.ami
    }
  }

}

run "test" {}
