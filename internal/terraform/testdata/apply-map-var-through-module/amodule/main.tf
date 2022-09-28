variable "amis" {
  type = map(string)
}

resource "null_resource" "noop" {}

output "amis_out" {
  value = var.amis
}
