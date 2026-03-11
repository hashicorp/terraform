
# This fixture is used by TestInputVariableValidationWithProviderFunction to
# verify that provider-defined functions can be called inside a validation
# condition expression.
#
# The mock test provider exposes a single function "upper" that converts a
# string to upper-case; the validation here checks that the given value
# equals "HELLO" when converted to upper-case.

required_providers {
  testing = {
    source = "terraform.io/builtin/testing"
  }
}

provider "testing" "main" {}

variable "foo" {
  type = string

  validation {
    condition     = provider::testing::upper(var.foo) == "HELLO"
    error_message = "Value must equal 'hello' (case-insensitive)."
  }
}
