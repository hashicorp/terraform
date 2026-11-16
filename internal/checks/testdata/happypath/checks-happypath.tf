resource "null_resource" "a" {
  lifecycle {
    precondition {
      condition     = null_resource.no_checks == ""
      error_message = "Impossible."
    }
    precondition {
      condition     = null_resource.no_checks == ""
      error_message = "Also impossible."
    }
    postcondition {
      condition     = null_resource.no_checks == ""
      error_message = "Definitely not possible."
    }
  }
}

resource "null_resource" "no_checks" {
}

module "child" {
  source = "./child"
}

output "a" {
  value = null_resource.a.id

  precondition {
    condition     = null_resource.a.id != ""
    error_message = "A has no id."
  }
}

check "check" {
  assert {
    condition = null_resource.a.id != ""
    error_message = "check block: A has no id"
  }
}
