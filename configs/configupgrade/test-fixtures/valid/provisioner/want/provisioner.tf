variable "login_username" {
}

resource "aws_instance" "foo" {
  connection {
    host = coalesce(self.public_ip, self.private_ip)
    type = "ssh"
    user = var.login_username
  }

  provisioner "test" {
    commands = ["a", "b", "c"]

    when       = create
    on_failure = fail

    connection {
      host = coalesce(self.public_ip, self.private_ip)
      type = "winrm"
      user = var.login_username
    }
  }
}
