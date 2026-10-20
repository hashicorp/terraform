variable "input" {
  type = string
}

ephemeral "testing_resource" "resource" {}

output "value" {
  value     = ephemeral.testing_resource.resource.value
  ephemeral = true

  precondition {
    condition     = var.input == "a-value"
    error_message = "The ephemeral component must receive the output of component.a."
  }
}
