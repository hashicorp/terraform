variable "string" {
  default = "foo"
  type    = string
}

variable "list" {
  default = []
  type    = list(string)
}

variable "map" {
  default = {}
  type    = map(string)
}

resource "aws_instance" "bar" {
  string  = var.string
  list    = var.list
  map     = var.map
}
