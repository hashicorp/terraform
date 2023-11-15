resource "null_resource" "foo" {
  provisioner "local-exec" {
    command = "echo Hello World!"
  }
}
