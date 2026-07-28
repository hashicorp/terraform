variable "foo" {
    default = "default-value"
}

resource "test_instance" "foo" {
    value = var.foo
}
