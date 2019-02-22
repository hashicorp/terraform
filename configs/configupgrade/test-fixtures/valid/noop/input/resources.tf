
resource "test_instance" "example" {
  connection {
    host = "127.0.0.1"
  }
  provisioner "test" {
    connection {
      host = "127.0.0.2"
    }
  }
}
