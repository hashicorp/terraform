locals {
  value = "local"
}

resource "aws_instance" "foo" {
    provisioner "shell" {
        command  = "${local.value}"
        when = "create"
    }
    provisioner "shell" {
        command  = "${local.value}"
        when = "destroy"
    }
}
