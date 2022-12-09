resource "test_instance" "foo" {
  count = 3
}

resource "test_load_balancer" "foo" {
  instances = test_instance.foo.*.id
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
