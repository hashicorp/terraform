variable "s" {
  type = string
}

variable "l" {
  type = list(string)

  default = []
}

variable "m" {
  type = map(string)

  default = {}
}
