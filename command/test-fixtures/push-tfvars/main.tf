variable "foo" {}
variable "bar" {}

variable "baz" {
    type = "map"
    default = {
        "A" = "a"
        "B" = "b"
    }
}

variable "fob" {
    type = "list"
    default = ["a", "b", "c"]
}

resource "test_instance" "foo" {}

atlas {
    name = "foo"
}
