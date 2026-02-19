
variable "input" {

  type = object({
    required = string
    optional = optional(string)
    default = optional(string, "default")
  })

  default = {
    required = "required"
  }

}

resource "test_resource" "resource" {
  value = var.input.default
}

output "computed" {
  value = test_resource.resource.value
}

output "input" {
  value = var.input
}
