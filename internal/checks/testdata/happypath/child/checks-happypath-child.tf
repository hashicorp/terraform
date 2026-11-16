resource "null_resource" "b" {
  lifecycle {
    precondition {
      condition     = self.id == ""
      error_message = "Impossible."
    }
  }
}

resource "null_resource" "c" {
  count = 2

  lifecycle {
    postcondition {
      condition     = self.id == ""
      error_message = "Impossible."
    }
  }
}

output "b" {
  value = null_resource.b.id

  precondition {
    condition     = null_resource.b.id != ""
    error_message = "B has no id."
  }
}

