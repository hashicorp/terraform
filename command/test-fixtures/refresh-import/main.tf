variable "foo" {
    default = "bar"
}

provider "test" {
    value = "${var.foo}"
}

resource "test_instance" "foo" {}
