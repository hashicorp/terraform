terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "test_instance" "a" {
  count = 0
}

resource "test_instance" "b" {
  value = test_instance.a[0].value
}
