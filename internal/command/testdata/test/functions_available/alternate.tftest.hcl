variables {
  input = jsonencode({key:"value"})
}

run "test" {
  assert {
    condition = jsondecode(test_resource.resource.value).key == "value"
    error_message = "wrong value"
  }
}
