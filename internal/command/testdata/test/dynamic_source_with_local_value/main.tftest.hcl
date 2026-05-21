run "validate_dynamic_module" {
  assert {
    condition     = module.mod.value == "bar"
    error_message = "expected bar from dynamically sourced module"
  }
}
