resource "test_instance" "foo" {
    ami = "${var.nope}"
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
