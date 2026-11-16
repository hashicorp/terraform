resource "test" "test" {
  lifecycle {
    precondition {
      condition     = path.module != ""
      error_message = "Must be true."
    }
    postcondition {
      condition     = path.module != ""
      error_message = "Must be true."
    }
  }
}

data "test" "test" {
  lifecycle {
    precondition {
      condition     = path.module != ""
      error_message = "Must be true."
    }
    postcondition {
      condition     = path.module != ""
      error_message = "Must be true."
    }
  }
}

output "test" {
  value = ""

  precondition {
    condition     = path.module != ""
    error_message = "Must be true."
  }
}
