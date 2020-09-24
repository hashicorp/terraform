variable "y" {
  type = string
}

resource "test_instance" "t" {
  foo = var.y
}
