resource "aws_security_group" "firewall" {
  provisioner "local-exec" {
    command = "echo hello"
    when    = "destroy"
  }
}
