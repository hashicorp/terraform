# This is an example test file from a use case requested by a user. We only
# refer to alternate modules and not the main configuration. This means we
# shouldn't have to provide any data for the main configuration.

run "first" {
  module {
    source = "./example"
  }

  variables {
    input = "start"
  }
}

run "second" {
  module {
    source = "./example"
  }

  variables {
    input = "update"
  }
}
