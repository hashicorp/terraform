
variable "input" {
  type = object({
    one = string,
    two = string,
  })
}

resource "test_resource" "resource" {
  value = var.input.two
}

output "one" {
  value = test_resource.resource.id
}

output "two" {
  value = test_resource.resource.value
}
