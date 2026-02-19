variable "foo" {
  type      = string
  ephemeral = true
}

variable "bar" {
  type      = string
  default   = null
  ephemeral = true
}

variable "unused" {
  type = map(string)
  default = null
}

resource "test_instance" "foo" {
  ami = "bar"
}
