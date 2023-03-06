terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "test_resource" "for_output" {
  required = "val"
}

output "object" {
  value = test_resource.for_output
}
