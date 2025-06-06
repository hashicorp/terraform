variables {
  default = "double"

  ref_one = var.default
  ref_two = run.secondary.value
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

run "tertiary" {
  variables {
    input_one = var.ref_one
    input_two = var.ref_two
  }

  assert {
    condition = output.value == "double - double - ${var.global}"
    error_message = "bad concatenation"
  }
}
