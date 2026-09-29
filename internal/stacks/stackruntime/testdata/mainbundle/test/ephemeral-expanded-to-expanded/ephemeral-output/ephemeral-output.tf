variable "id" {
  type = string
}

resource "testing_resource" "resource" {
  count = var.id == "a" ? 1 : 0
  id    = "out-a"
}

ephemeral "testing_resource" "resource" {}

output "value" {
  value     = ephemeral.testing_resource.resource.value
  ephemeral = true
}
