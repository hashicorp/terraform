
terraform {
  experiments = [smoke_tests]
}

locals {
  a = "a"
}

smoke_test "just_postcondition" {
  postcondition {
    condition     = local.a == "a"
    error_message = "Wrong A."
  }
}

smoke_test "just_precondition" {
  precondition {
    condition     = local.a == "a"
    error_message = "Wrong A."
  }
}

smoke_test "just_data" {
  data "foo" "bar" {
  }
}

smoke_test "everything" {
  precondition {
    condition     = local.a == "a"
    error_message = "Wrong A."
  }

  data "foo" "bar" {
  }

  postcondition {
    condition     = data.foo.bar.okay
    error_message = "Not okay."
  }
}

smoke_test "many_things" {
  precondition {
    condition     = local.a == "a"
    error_message = "Wrong A."
  }

  precondition {
    condition     = local.a == "a"
    error_message = "Wrong A again."
  }

  data "foo" "bar" {
  }

  data "foo" "baz" {
  }

  postcondition {
    condition     = data.foo.bar.okay
    error_message = "Not okay."
  }

  postcondition {
    condition     = data.foo.bar.okay
    error_message = "Not okay again."
  }
}
