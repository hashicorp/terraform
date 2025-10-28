
variable "input" {
  type = string
  ephemeral = true
}

resource "testing_write_only_resource" "resource" {
  id = "8453e0fa5aa2"
  write_only = var.input
}
