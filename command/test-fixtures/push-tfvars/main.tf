variable "foo" {}

variable "bar" {}

variable "baz" {
  type = "map"

  default = {
    "A"    = "a"
    interp = "${file("t.txt")}"
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
