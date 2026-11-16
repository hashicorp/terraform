
override_resource {
  target = data.aws_instance.target
  values = {}
}

override_resource {
  target = module.child
  values = {}
}

run "test" {}
