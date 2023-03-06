terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "test_object" "b" {
  test_string = "changed"
}
