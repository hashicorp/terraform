
run "test" {
  command = apply
  plan_options {
    mode = refresh-only
  }
  assert {
    condition = test_resource.example.value == "bar"
    error_message = "wrong value"
  }
}
