
variable "input" {
  type = map(object({
    output = string
  }))
}

resource "testing_resource" "main" {
  for_each = var.input
  id = each.key
  value = each.value.output
}
