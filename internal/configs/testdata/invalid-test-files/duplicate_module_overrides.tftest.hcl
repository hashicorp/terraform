
override_module {
  target = module.child
  outputs = {}
}

override_module {
  target = module.child
  outputs = {}
}

run "test" {
  override_module {
    target = module.child
    outputs = {}
  }

  override_module {
    target = module.child
    outputs = {}
  }
}
