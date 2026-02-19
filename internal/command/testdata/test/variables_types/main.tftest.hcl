run "variables" {

  # This run block requires the following variables to have been defined as
  # command line arguments.

  assert {
    condition = var.number_input == 0
    error_message = "bad number value"
  }

  assert {
    condition = var.string_input == "Hello, world!"
    error_message = "bad string value"
  }

  assert {
    condition = var.list_input == tolist(["Hello", "world"])
    error_message = "bad list value"
  }
}
