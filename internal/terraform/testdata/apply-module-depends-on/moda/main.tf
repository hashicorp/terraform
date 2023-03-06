terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "test_instance" "a" {
  num = 5
}

data "test_data_source" "a" {
  foo = "a"
}

output "out" {
  value = data.test_data_source.a.id
}
