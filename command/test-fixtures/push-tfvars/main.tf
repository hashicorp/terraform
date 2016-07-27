variable "foo" {}

variable "bar" {}

variable "baz" {
  type = "map"

  default = {
    "A"    = "a"
    "B"    = "b"
    interp = "${file("t.txt")}"
  }
}

variable "fob" {
  type    = "list"
  default = ["a", "b", "c", "quotes \"in\" quotes"]
}

resource "test_instance" "foo" {}

atlas {
  name = "foo"
}
