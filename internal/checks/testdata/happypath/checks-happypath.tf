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

terraform {
  experiments = [smoke_tests]
}

smoke_test "a" {
  precondition {
    condition     = null_resource.a.id != ""
    error_message = "A has no id."
  }

  data "foo" "bar" {
    a_id = null_resource.a.id
  }

  postcondition {
    condition     = data.foo.bar.okay
    error_message = "A is not okay."
  }
}
