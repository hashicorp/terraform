
variable "a" {
  type = string
}

resource "test_thing" "a" {
  arg = var.a
}
