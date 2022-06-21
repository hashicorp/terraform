data "example" "example" {
  foo = 5

  lifecycle {
    precondition {
      condition     = data.example.example.foo == 5 # ERROR: Invalid reference in precondition
      error_message = "Must be five."
    }
    postcondition {
      condition     = self.foo == 5
      error_message = "Must be five, but is ${data.example.example.foo}." # ERROR: Invalid reference in postcondition
    }
  }
}

resource "example" "example" {
  foo = 5

  lifecycle {
    precondition {
      condition     = example.example.foo == 5 # ERROR: Invalid reference in precondition
      error_message = "Must be five."
    }
    postcondition {
      condition     = self.foo == 5
      error_message = "Must be five, but is ${example.example.foo}." # ERROR: Invalid reference in postcondition
    }
  }
}
