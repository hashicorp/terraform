variable "list" {
  type = list(string)
}

resource "aws_instance" "bar" {
  count = var.list[0]
}
