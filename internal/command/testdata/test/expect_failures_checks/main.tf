
variable "input" {
  type = string
}

resource "test_resource" "resource" {
  value = var.input
}

check "expected_to_fail" {
  assert {
    condition = test_resource.resource.value != var.input
    error_message = "this really should fail"
  }
}
