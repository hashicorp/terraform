
variables {
  input = {
    organization_name = var.org_name
  }
}

run "execute" {
  assert {
    condition     = output.value.organization_name == "my-org"
    error_message = "bad output value"
  }
}
