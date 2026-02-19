resource "null_resource" "foo" {
  provisioner "local-exec" {
    command = "exit 125"
  }
}
