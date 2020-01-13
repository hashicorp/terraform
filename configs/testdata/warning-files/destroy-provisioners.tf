locals {
  user = "name"
}

resource "null_resource" "a" {
  connection {
    host = self.hostname
    user = local.user # WARNING: External references from destroy provisioners are deprecated
  }

  provisioner "remote-exec" {
    when = destroy
    index = count.index
    key = each.key
    dir = path.module
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
      user = local.user # WARNING: External references from destroy provisioners are deprecated
    }

    command = "echo ${local.name}" # WARNING: External references from destroy provisioners are deprecated
  }
}
