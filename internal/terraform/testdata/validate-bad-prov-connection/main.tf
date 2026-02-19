resource "aws_instance" "foo" {
  provisioner "shell" {
    test_string = "test"
    connection {
      user = "test"
    }
  }
}
