override_resource {
  target  = test_resource.primary
  values = "should be an object" // invalid
}

variables {
  instances = 2
}

run "test" {
  # We won't even execute this, as the configuration isn't valid.
}
