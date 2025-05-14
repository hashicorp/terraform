override_module {
  target  = module.child[1]
  outputs = "should be an object"
}

variables {
  instances = 3
  child_instances = 1
}

run "test" {
  # We won't even execute this, as the configuration isn't valid.
}
