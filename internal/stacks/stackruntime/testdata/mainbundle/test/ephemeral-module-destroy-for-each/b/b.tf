variable "name" {
  type = string
}

resource "testing_resource" "b" {
  id = "b-${var.name}"
}
