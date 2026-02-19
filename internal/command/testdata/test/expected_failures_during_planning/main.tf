
variable "input" {
  type = string

  validation {
    condition = strcontains(var.input, "a")
    error_message = "input must contain the character 'a'"
  }
}

resource "test_resource" "resource" {
  value = var.input

  lifecycle {
    postcondition {
      condition = strcontains(self.value, "b")
      error_message = "input must contain the character 'b'"
    }
  }
}

check "cchar" {
  assert {
    condition = strcontains(test_resource.resource.value, "c")
    error_message = "input must contain the character 'c'"
  }
}

output "output" {
  value = test_resource.resource.value

  precondition {
    condition = strcontains(test_resource.resource.value, "d")
    error_message = "input must contain the character 'd'"
  }
}

