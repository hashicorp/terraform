resource "test" "test" {
  lifecycle {
    precondition {
      condition     = true # ERROR: Invalid precondition expression
      error_message = "Must be true."
    }
    postcondition {
      condition     = true # ERROR: Invalid postcondition expression
      error_message = "Must be true."
    }
  }
}

data "test" "test" {
  lifecycle {
    precondition {
      condition     = true # ERROR: Invalid precondition expression
      error_message = "Must be true."
    }
    postcondition {
      condition     = true # ERROR: Invalid postcondition expression
      error_message = "Must be true."
    }
  }
}

output "test" {
  value = ""
 
  precondition {
    condition     = true # ERROR: Invalid precondition expression
    error_message = "Must be true."
  }
}
