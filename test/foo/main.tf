resource "null_resource" "foo" {
  count = 2

  provisioner "local-exec" { command = "sleep ${count.index*3}" }

  //provisioner "local-exec" { command = "exit 1" }

  lifecycle { create_before_destroy = true }
}
