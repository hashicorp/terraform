resource "aws_instance" "b" {
  count    = length(var.ids)
  require_new = var.ids[count.index]
}

variable "ids" {
  type = list(string)
}
