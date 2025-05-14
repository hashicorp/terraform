override_module {
  target = module.child
}

override_module {
  outputs = {}
}

override_module {
  target = module.other
  values = {}
}

run "test" {}
