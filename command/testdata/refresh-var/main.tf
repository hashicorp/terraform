variable "foo" {}

provider "test" {
    value = "${var.foo}"
}

resource "test_instance" "foo" {}
