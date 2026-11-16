variable "password" {
  type      = string
  ephemeral = true
}

resource "aws_instance" "foo" {
  connection {
    host     = "localhost"
    type     = "telnet"
    user     = "superuser"
    port     = 2222
    password = "password"
  }

  provisioner "shell" {
    command = "echo ${var.password} > secrets"
  }
}
