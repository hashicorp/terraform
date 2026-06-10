variable "foo" {
    default = "default-value"
}

provider "test" {
    value = var.foo
}

resource "test_instance" "foo" {}
