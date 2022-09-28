variable "number" {
  default = 3
}

variable "string" {
  default = "Hello, World"
}

variable "map" {
  type = map(string)
  default = {
    "foo" = "bar",
    "baz" = "bat",
  }
}

locals {
  result = length(var.list)
}

variable "list" {
  type    = list(string)
  default = ["red", "orange", "yellow", "green", "blue", "purple"]
}

resource "test_resource" "example" {
  for_each = var.map
  name     = each.key
  tag      = each.value
}

module "child" {
  source = "./child"
  list   = var.list
}

output "result" {
  value = module.child.result
}
