# This file starts with a resource and a required providers block, and should
# end up with the full required providers configuration. This file is chosen
# to keep the required providers block because its file name is "providers.tf".
resource bar_instance b {}

terraform {
  required_providers {
    bar = {
      source = "registry.acme.corp/acme/bar"
    }
  }
}
