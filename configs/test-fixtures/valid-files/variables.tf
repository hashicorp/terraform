
variable "foo" {
}

variable "bar" {
  default = "hello"
}

variable "baz" {
  type = list
}

variable "bar-baz" {
  default = []
  type    = list
}

variable "cheeze_pizza" {
  description = "Nothing special"
}

variable "π" {
  default = 3.14159265359
}
