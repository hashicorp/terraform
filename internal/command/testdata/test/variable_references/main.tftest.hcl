variables {
  default = "double"
}

run "primary" {
  variables {
    input_one = var.default
    input_two = var.default
  }

  assert {
    condition = test_resource.resource.value == "${var.default} - ${var.input_two}"
    error_message = "bad concatenation"
  }
}

run "secondary" {
  variables {
    input_one = var.default
    input_two = var.global # This test requires this passed in as a global var.
  }

  assert {
    condition = test_resource.resource.value == "double - ${var.global}"
    error_message = "bad concatenation"
  }
}
