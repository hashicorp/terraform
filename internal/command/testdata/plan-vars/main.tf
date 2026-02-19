variable "foo" {}

resource "test_instance" "foo" {
    value = "${var.foo}"
}
