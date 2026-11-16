mock_provider "aws" {
  override_resource {
    target = aws_instance.test
    values = {}
  }

  override_resource {
    target = aws_instance.test
    values = {}
  }
}

override_resource {
  target = aws_instance.test
  values = {}
}

override_resource {
  target = aws_instance.test
  values = {}
}

run "test" {
  override_resource {
    target = aws_instance.test
    values = {}
  }

  override_resource {
    target = aws_instance.test
    values = {}
  }
}
