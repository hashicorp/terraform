resource "aws_security_group" "firewall" {
  lifecycle {
    create_before_destroy = true
    prevent_destroy = true
    ignore_changes = [
      description,
    ]
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

resource "aws_instance" "web" {
  count = 2
  ami = "ami-1234"
  security_groups = [
    "foo",
    "bar",
  ]

  network_interface {
    device_index = 0
    description = "Main network interface"
  }

  depends_on = [
    aws_security_group.firewall,
  ]
}

resource "aws_instance" "depends" {
  lifecycle {
    replace_triggered_by = [ aws_instance.web[1], aws_security_group.firewall.id ]
  }
}
