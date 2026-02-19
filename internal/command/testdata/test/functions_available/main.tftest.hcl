
run "test" {
  variables {
    input = jsonencode({key:"value"})
  }

  assert {
    condition = jsondecode(test_resource.resource.value).key == "value"
    error_message = "wrong value"
  }
}
