variable "input" {}

resource "test_object" "foo" {
  test_string = var.input
}

output "output" {
  value = test_object.foo.id
}
