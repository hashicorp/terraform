
override_data {
  target = aws_instance.target
  values = {}
}

override_data {
  target = module.child
  values = {}
}

run "test" {}
