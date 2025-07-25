variable "defer" {
  type = bool
}

resource "test_resource" "resource" {
  defer = var.defer
}
