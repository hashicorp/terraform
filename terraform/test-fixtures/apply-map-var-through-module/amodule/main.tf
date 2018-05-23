variable "amis" {
    type = "map"
}

resource "null_resource" "noop" {}

output "amis_out" {
  value = "${var.amis}"
}
