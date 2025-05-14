
override_data {
  target = data.aws_instance.target
}

override_data {
  values = {}
}

run "test" {}
