variable "foo" {}

variable "bar" {}

variable "baz" {
  type = "map"

  default = {
    "A"    = "a"
  }
}

variable "fob" {
  type    = "list"
  default = ["a", "quotes \"in\" quotes"]
}

resource "test_instance" "foo" {}

atlas {
  name = "foo"
}
