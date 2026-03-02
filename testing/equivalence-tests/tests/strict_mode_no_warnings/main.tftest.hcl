# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: BUSL-1.1

run "test" {
  command = plan

  variables {
    input = "Hello, world!"
  }

  assert {
    condition     = tfcoremock_simple_resource.resource.string == "Hello, world!"
    error_message = "expected string to be Hello, world!"
  }
}
