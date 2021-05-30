resource "null_resource" "a" {
  provisioner "local-exec" {
    command = "echo HelloProvisioner"
  }
}
