mock_provider "aws" {
  override_provisioner {
    target = aws_instance.test
    values = {}
  }

  override_provisioner {
    target = aws_instance.test
    values = {}
  }
}

override_provisioner {
  target = aws_instance.test
  values = {}
}

override_provisioner {
  target = aws_instance.test
  values = {}
}

run "test" {
  override_provisioner {
    target = aws_instance.test
    values = {}
  }

  override_provisioner {
    target = aws_instance.test
    values = {}
  }
}
