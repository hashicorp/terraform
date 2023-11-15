
variables {
  # config_free isn't defined in the config, but we'll
  # still let users refer to it within the assertions.
  config_free = "Hello, world!"
}

run "applies_defaults" {
  assert {
    condition     = var.input == var.config_free
    error_message = "should have applied default value"
  }
}
