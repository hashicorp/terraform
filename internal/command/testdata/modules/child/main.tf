resource "test_instance" "test" {
}
output "myoutput" {
  value = "bar"
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
