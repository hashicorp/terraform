
locals {
  my_value = "Hello, world!"
}

resource "test_resource" "example" {
    value = local.my_value
}
