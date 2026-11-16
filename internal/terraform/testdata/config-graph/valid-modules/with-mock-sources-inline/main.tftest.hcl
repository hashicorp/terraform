
mock_provider "aws" {
  source = "./testing/aws"

  mock_resource "aws_s3_bucket" {
    defaults = {
      arn = "aws:s3:::bucket"
    }
  }
}

run "test" {}
