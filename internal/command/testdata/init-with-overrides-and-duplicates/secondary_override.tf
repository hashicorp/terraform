terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
      version = "1.0.0"
    { // This typo is deliberate, we want to test that the parser can handle it.
  }
}

resource "test_instance" "bar" {
  ami = "override"
}
