resource "test_instance" "foo" {
  connection {
    type = "ssh"
    host = "${self.private_ip}"
  }

  provisioner "test" {
    commands = "${list("a", "b", "c")}"

    when       = "create"
    on_failure = "fail"

    connection {
      type = "winrm"
      host = "${self.public_ip}"
    }
  }
}
