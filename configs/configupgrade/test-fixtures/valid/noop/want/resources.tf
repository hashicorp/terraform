
resource "test_instance" "example" {
  connection {
    host = "127.0.0.1"
  }
  provisioner "local-exec" {
    connection {
      host = "127.0.0.2"
    }
  }
}
