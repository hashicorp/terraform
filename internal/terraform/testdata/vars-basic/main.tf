variable "a" {
  default = "foo"
  type    = string
}

variable "b" {
  default = []
  type    = list(string)
}

variable "c" {
  default = {}
  type    = map(string)
}
