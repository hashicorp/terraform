variable "list" {
  default = ["1", "2"]
}

resource "aws_instance" "a" {
  count = length(var.list)
}

locals {
  ids = aws_instance.a[*].id
}

module "empty" {
  source = "./mod"
  input = zipmap(var.list, local.ids)
}

output "out" {
  value = aws_instance.a[*].id
}
