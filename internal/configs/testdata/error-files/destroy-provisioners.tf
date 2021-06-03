locals {
  user = "name"
}

resource "null_resource" "a" {
  connection {
    host = self.hostname
    user = local.user # ERROR: Invalid reference from destroy provisioner
  }

  provisioner "remote-exec" {
    when = destroy
    index = count.index
    key = each.key

    // path and terraform values are static, and do not create any
    // dependencies.
    dir = path.module
    workspace = terraform.workspace
  }
}

resource "null_resource" "b" {
  connection {
    host = self.hostname
    # this is OK since there is no destroy provisioner
    user = local.user
  }

  provisioner "remote-exec" {
  }
}

resource "null_resource" "b" {
  provisioner "remote-exec" {
    when = destroy
    connection {
      host = self.hostname
      user = local.user # ERROR: Invalid reference from destroy provisioner
    }

    command = "echo ${local.name}" # ERROR: Invalid reference from destroy provisioner
  }
}
