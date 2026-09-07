terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

# Open an ephemeral resource to expose its value.
ephemeral "test_ephemeral_resource" "data" {}

# Surface the ephemeral value as an ephemeral output so the test can assert on it.
output "ephemeral_value" {
  value     = ephemeral.test_ephemeral_resource.data.value
  ephemeral = true
}
