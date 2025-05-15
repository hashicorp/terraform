action "provider_reboot" "powercycle" {
    method = "biggest_hammer"  // drop it in the ocean, i presume
}

resource "aws_security_group" "firewall" {
  lifecycle {
    create_before_destroy = true
    prevent_destroy = true
    ignore_changes = [
      description,
    ]
    action_trigger {
        events = [after_create, after_update]
        condition = has_changed(self.environment)
        actions = [action.provider_reboot.powercycle]
    }
  }

  connection {
    host = "127.0.0.1"
  }

  provisioner "local-exec" {
    command = "echo hello"

    connection {
      host = "10.1.2.1"
    }
  }

  provisioner "local-exec" {
    command = "echo hello"
  }
}
