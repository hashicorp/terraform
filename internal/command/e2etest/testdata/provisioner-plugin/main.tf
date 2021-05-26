resource "null_resource" "a" {
  provisioner "test" {
    command = "echo HelloProvisioner"
  }
}
