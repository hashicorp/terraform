
provider "aws" {}

override_data {
  target = data.aws_secretsmanager_secret.creds
  values = {
    arn = "aws:secretsmanager"
  }
}

run "test" {
  override_resource {
    target = aws_instance.first
    values = {
      arn = "aws:instance:first"
    }
  }

  override_data {
    target = data.aws_secretsmanager_secret.creds
    values = {
      arn = "aws:secretsmanager"
    }
  }
}
