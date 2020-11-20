resource "test" "test" {
  lifecycle {
    precondition { # ERROR: Preconditions are experimental
      condition     = path.module != ""
      error_message = "Must be true."
    }
    postcondition { # ERROR: Postconditions are experimental
      condition     = path.module != ""
      error_message = "Must be true."
    }
  }
}

data "test" "test" {
  lifecycle {
    precondition { # ERROR: Preconditions are experimental
      condition     = path.module != ""
      error_message = "Must be true."
    }
    postcondition { # ERROR: Postconditions are experimental
      condition     = path.module != ""
      error_message = "Must be true."
    }
  }
}

output "test" {
  value = ""

  precondition { # ERROR: Preconditions are experimental
    condition     = path.module != ""
    error_message = "Must be true."
  }
}
