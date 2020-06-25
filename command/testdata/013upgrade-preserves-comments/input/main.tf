resource foo_instance a {}
resource bar_instance b {}

terraform {
  # Provider requirements go here
  required_providers {
    # Pin bar to this version
    bar = "0.5.0"
    # An explicit requirement
    baz = {
      # Comment inside the block should stay
      source = "foo/baz"
    }
    # Foo is required
    foo = {
      # This comment sadly won't make it
      version = "1.0.0"
    }
  }
}
