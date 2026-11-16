
override_resource {
  target = aws_instance.target
}

override_resource {
  values = {}
}

run "test" {}
