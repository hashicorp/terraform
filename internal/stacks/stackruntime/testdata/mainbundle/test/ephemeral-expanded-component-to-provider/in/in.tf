variable "id" {
  type = string
}

resource "testing_resource" "resource" {
  id = var.id
}
