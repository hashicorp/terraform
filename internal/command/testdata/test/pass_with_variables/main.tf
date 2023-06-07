variable "input" {
  type = string
}

resource "test_instance" "foo" {
  ami = var.input
}
