variable "foo" {}

provider "test" {
    foo = "${var.foo}"
}

resource "test_instance" "foo" {
}
