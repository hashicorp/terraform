terraform {
required_providers {
    testing = {
      source = "hashicorp/testing"
    }
  }
}

variable "name" {
  type = string
}
variable "list" {
  type = list(string)
}
variable "map" {
  type = map(string)
}

resource "testing_resource" "resource" {
  id = var.name
  value = "foo"
}

data "testing_data_source" "data_source" {
  id = var.name
  depends_on = [testing_resource.resource]
}

output "bar" {
  value = "${var.name}-${data.testing_data_source.data_source.value}"
}

output "list" {
  value = concat(var.list, ["${data.testing_data_source.data_source.value}"])
}

output "map" {
  value = merge(var.map, { "value" = data.testing_data_source.data_source.value })
}
