override_module {
  target = module.child[1]
}

variables {
  instances = 3
  child_instances = 1
}

run "test" {
  # Just want to make sure things don't crash with missing `outputs` attribute.
}
