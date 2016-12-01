variable "username" {
  default = "bob"
}

data "template_file" "user" {
  template = "$${USERNAME}"
  vars {
    USERNAME = "${var.username}"
  }
  provisioner "local-exec" {
    command = "echo ${self.rendered} > user.txt"
  }
}
