variable "input" {
  type = string
}

resource "test_resource" "foobar" {
  # No ID set here
  # We should be able to assert about its value as it will be loaded from state
  # by the backend block in the run block
  value = var.input
}

output "test_resource_id" {
  value = test_resource.foobar.id
}

output "supplied_input_value" {
  value = var.input
}
