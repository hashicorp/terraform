resource "example" "example" {
  foo = 5

  lifecycle {
    precondition { # ERROR: Missing required argument
      error_message = "Can a check block fail without a condition?"
    }
    postcondition { # ERROR: Missing required argument
      error_message = "Do not try to pass the check; only realize that there is no check."
    }
  }
}
