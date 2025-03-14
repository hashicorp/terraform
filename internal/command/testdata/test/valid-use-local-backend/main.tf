# This test fixure allows conditional provisioning of a second resource.
# This is used to test which run blocks control state about long-lived 
# resources used in tests.

variable "input" {
  type = string
}

variable "provision_second_resource" {
  type    = bool
  default = false
}

output "supplied_input_value" {
  value = var.input
}

# This resource is 'long-lived' in tests; it should always be provisioned
# and is kept in state.
resource "test_resource" "a" {
  id    = "12345"
  value = var.input
}

# This resource is not 'long-lived' in tests; is only provisioned during tests
# when var.provision_second_resource is set to true. It should not be entering
# persisted state. 
resource "test_resource" "b" {
  count = var.provision_second_resource ? 1 : 0
  id    = "67890"
  value = var.input
}
