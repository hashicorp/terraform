
# NOTE: This fixture is used in a test that doesn't run a full Terraform plan
# operation, so the count and for_each expressions here can only be literal
# values and mustn't include any references or function calls.

module "single" {
  source = "./child"
}

module "count" {
  source = "./child"
  count  = 2
}

module "zero_count" {
  source = "./child"
  count  = 0
}

module "for_each" {
  source = "./child"
  for_each = {
    a = "A"
  }
}

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

resource "other" "single" {
}

module "fake_external" {
  # Our configuration fixture loader has a special case for a module call
  # named "fake_external" where it will mutate the source address after
  # loading to instead be an external address, so we can test rules relating
  # to crossing module boundaries.
  source = "./child"
}
