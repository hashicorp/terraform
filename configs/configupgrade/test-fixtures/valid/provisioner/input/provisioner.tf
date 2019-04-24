variable "login_username" {}

resource "aws_instance" "foo" {
  connection {
    user = "${var.login_username}"
  }

  provisioner "test" {
    commands = "${list("a", "b", "c")}"

    when       = "create"
    on_failure = "fail"

    connection {
      type = "winrm"
      user = "${var.login_username}"
    }
  }
}
