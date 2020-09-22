# This file starts with a resource and a required providers block, and should
# end up with just the resource.
resource foo_instance a {}

terraform {
  required_providers {
    foo = "1.0.0"
  }
}
