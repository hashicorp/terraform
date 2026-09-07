variable "input" {
  type = string
}

resource "test_resource" "foobar" {
  id = "12345"
  # Set deterministic ID because this fixture is for testing what happens when there's no prior state
  # i.e. this id will otherwise keep changing per test
  value = var.input
}

output "test_resource_id" {
  value = test_resource.foobar.id
}

output "supplied_input_value" {
  value = var.input
}
