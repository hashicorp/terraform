
locals {
  input = uuid() # using UUID to ensure that plan phase will return an unknown value
}

output "output" {
  value = local.input

  precondition {
    condition = local.input != ""
    error_message = "this should not fail during the apply phase"
  }
}
