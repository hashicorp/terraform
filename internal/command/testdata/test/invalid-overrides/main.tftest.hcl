
mock_provider "test" {
  override_resource {
    target = test_resource.absent_one
  }
}

override_resource {
  target = test_resource.absent_two
}

override_resource {
  target = module.setup.test_resource.absent_three
}

override_module {
  target = module.absent_four
}

override_resource {
  // This one only exists in the main configuration, but not the setup
  // configuration. We shouldn't see a warning for this.
  target = module.setup.test_resource.child_resource
}

override_resource {
  // This is the reverse, only exists if you load the setup module directly.
  // We shouldn't see a warning for this even though it's not in the main
  // configuration.
  target = test_resource.child_resource
}

run "setup" {
  module {
    source = "./setup"
  }

  override_resource {
    target = test_resource.absent_five
  }
}

run "test" {
  override_resource {
    target = module.setup.test_resource.absent_six
  }
}
