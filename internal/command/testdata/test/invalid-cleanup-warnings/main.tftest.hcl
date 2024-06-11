# main.tftest.hcl

run "test" {
  variables {
    input = "Hello, world!"
    validation = "Hello, world!"
  }
  assert {
    condition = test_resource.resource.value == "Hello, world!"
    error_message = "bad!"
  }
}

