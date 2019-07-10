
terraform {
  experiments = [
    resource_for_each,
  ]
}

resource "null_resource" "foo" {
  for_each = {}
}
