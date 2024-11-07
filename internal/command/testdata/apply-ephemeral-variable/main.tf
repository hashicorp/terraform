variable "foo" {
  type      = string
  ephemeral = true
}

variable "bar" {
  type      = string
  default   = null
  ephemeral = true
}

resource "test_instance" "foo" {
  ami = "bar"
}
