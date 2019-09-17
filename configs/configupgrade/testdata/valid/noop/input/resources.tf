
resource "test_instance" "example" {
  tags = {
    # Thingy thing
    name = "foo bar baz" # this is a terrible name
  }

  connection {
    host = "127.0.0.1"
  }
  provisioner "test" {
    connection {
      host = "127.0.0.2"
    }
  }
}
