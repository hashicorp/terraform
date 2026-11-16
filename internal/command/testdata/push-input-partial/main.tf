variable "foo" {}
variable "bar" {}

resource "test_instance" "foo" {}

atlas {
    name = "foo"
}
