terraform {
  required_providers {
    test = {
      source = "test"
    }
  }
}


output "value" {
  value = provider::test::is_true(true)
}
