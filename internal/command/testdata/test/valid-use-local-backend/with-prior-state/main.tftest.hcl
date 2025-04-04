run "setup_pet_name" {
  backend "local" {
    // Use default path
  }

  variables {
    input = "value-from-run-that-controls-backend"
  }
}

run "edit_input" {
  variables {
    input = "this-value-should-not-enter-state"
  }
}
