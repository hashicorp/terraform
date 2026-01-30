variable "name" {
  type = string
}

resource "testing_resource" "hello" {
  id = var.name
}
