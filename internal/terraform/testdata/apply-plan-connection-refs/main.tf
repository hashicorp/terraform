variable "msg" {
  default = "ok"
}

resource "test_instance" "a" {
  foo = "a"
}


resource "test_instance" "b" {
  foo = "b"
  provisioner "shell" {
   command = "echo ${var.msg}"
  }
  connection {
   host = test_instance.a.id
  }
}
