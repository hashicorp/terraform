
mock_provider "aws" {

  mock_resource "aws_instance" {
    defaults = {
      arn = "aws:instance"
    }
  }

  mock_data "aws_secretsmanager_secret" {}

  override_resource {
    target = aws_instance.second
    values = {}
  }

  override_data {
    target = data.aws_secretsmanager_secret.creds
    values = {
      arn = "aws:secretsmanager"
    }
  }
}

override_module {
  target = module.child
  outputs = {
    string = "testfile"
    number = -1
  }
}

run "test" {
  override_resource {
    target = aws_instance.first
    values = {
      arn = "aws:instance:first"
    }
  }

  override_module {
    target = module.child
    outputs = {
      string = "testrun"
      number = -1
    }
  }
}
