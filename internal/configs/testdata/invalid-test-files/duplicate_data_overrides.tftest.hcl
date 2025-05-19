mock_provider "aws" {
  override_data {
    target = data.aws_instance.test
    values = {}
  }

  override_data {
    target = data.aws_instance.test
    values = {}
  }
}

override_data {
  target = data.aws_instance.test
  values = {}
}

override_data {
  target = data.aws_instance.test
  values = {}
}

run "test" {
  override_data {
    target = data.aws_instance.test
    values = {}
  }

  override_data {
    target = data.aws_instance.test
    values = {}
  }
}
