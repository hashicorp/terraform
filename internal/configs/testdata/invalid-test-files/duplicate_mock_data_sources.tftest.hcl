mock_provider "aws" {

  mock_data "aws_instance" {
    defaults = {}
  }

  mock_data "aws_instance" {
    defaults = {}
  }

}

run "test" {}
