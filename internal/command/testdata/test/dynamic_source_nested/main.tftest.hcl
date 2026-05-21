run "validate_nested_dynamic_module" {
  assert {
    condition     = module.parent.value == "from_child"
    error_message = "expected from_child from nested dynamically sourced module"
  }
}
