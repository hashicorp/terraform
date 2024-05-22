# main.tftest.hcl

run "test" {
  variables {
    input = "Hello, world!"
    validation = "Hello, world!"
  }
  assert {
    condition = test_resource.resource.value == var.validation
    error_message = "bad!"
  }
}

