terraform {
    backend "local" {}
}

output "foo" {
  value = "bar"
}
