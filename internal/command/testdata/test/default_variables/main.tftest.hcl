
run "applies_defaults" {
  assert {
    condition     = var.input == "Hello, world!"
    error_message = "should have applied default value"
  }
}
