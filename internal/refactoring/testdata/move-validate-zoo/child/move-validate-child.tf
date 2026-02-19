
# NOTE: This fixture is used in a test that doesn't run a full Terraform plan
# operation, so the count and for_each expressions here can only be literal
# values and mustn't include any references or function calls.

resource "test" "single" {
}

resource "test" "count" {
  count = 2
}

resource "test" "zero_count" {
  count = 0
}

resource "test" "for_each" {
  for_each = {
    a = "A"
  }
}
