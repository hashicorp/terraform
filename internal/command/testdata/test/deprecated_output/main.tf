variable "name" {
  type    = string
  default = "example"
}

resource "test_resource" "foo" {
  value = var.name
}

output "deprecated_name" {
  value      = test_resource.foo.value
  deprecated = "This output is deprecated!"
}
