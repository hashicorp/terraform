# This run block contains the backend block, so it controls the state saved there
run "this_updates_state" {
  backend "local" {
    # Defaults to storing state in the working directory as terraform.tfstate
  }

  variables {
    input = "value-from-run-that-controls-backend"
  }

  assert {
    condition     = test_resource.a.output == "value-from-run-that-controls-backend"
    error_message = "test_resource.a.output value should match the input var"
  }
}

# This run block uses internal state loaded from the backend block,
# but changes made via this block (e.g. setting var.provision_second_resource = true)
#  do not affect the persisted state.
run "this_does_not_update_state" {
  variables {
    input                     = "this-value-should-not-enter-state"
    provision_second_resource = true
  }

  assert {
    condition     = test_resource.b.output == "this-value-should-not-enter-state"
    error_message = "test_resource.b should be provisioned, and have an output value that matches the input var"
  }
}
