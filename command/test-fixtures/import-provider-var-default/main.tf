variable "foo" {}

provider "test" {
    foo = "${var.foo}"
}
