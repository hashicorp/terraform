module "mod" {
  source = "./mod"
}

locals {
  value = "${module.mod.value}"
}

resource "aws_instance" "foo" {
    provisioner "shell" {
        command  = "${local.value}"
        when = "destroy"
    }
}

module "mod2" {
  source = "./mod2"
  value = "${module.mod.value}"
}
