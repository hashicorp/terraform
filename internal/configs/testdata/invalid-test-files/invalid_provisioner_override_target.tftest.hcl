override_provisioner {
  target = data.aws_instance.target
  values = {}
}

override_provisioner {
  target = module.child
  values = {}
}

run "test" {}
