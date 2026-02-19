mock_provider "aws" {

  mock_resource "aws_instance" {
    defaults = {}
  }

  mock_resource "aws_instance" {
    defaults = {}
  }

}

run "test" {}
