variable "instances" {
  type = number
}

variable "prefix" {
  type = string
}

resource "null_resource" "pet" {
  count = var.instances
}

output "pet_ids" {
  value = null_resource.pet[*].id
}
