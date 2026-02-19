variable "foo" {}

resource "test_instance" "foo" {
    foo = var.foo
}
