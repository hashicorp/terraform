resource "null_resource" "test" {
  provisioner "habitat" {} # ERROR: The "habitat" provisioner has been removed
}
