variable "param" {}

resource "null_resource" "n" {}

module "bottom" {
  source       = "./bottom"
  bottom_param = "${var.param}"
}
