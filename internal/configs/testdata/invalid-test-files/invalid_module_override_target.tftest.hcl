
override_module {
  target = aws_instance.target
  outputs = {}
}

override_module {
  target = data.aws_instance.target
  outputs = {}
}

run "test" {}
