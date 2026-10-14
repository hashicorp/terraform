variable "id" {
  type    = string
  default = "default"
}

ephemeral "testing_resource" "resource" {}

output "value" {
  value     = "${var.id}-${ephemeral.testing_resource.resource.value}"
  ephemeral = true
}
