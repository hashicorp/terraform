variable bottom_param {}

resource "null_resource" "bottom" {
  value = "${var.bottom_param}"
}
