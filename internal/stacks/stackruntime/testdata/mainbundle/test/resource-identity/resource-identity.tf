variable "name" {
    type = string
}

resource "testing_resource_with_identity" "hello" {
    id = var.name
}