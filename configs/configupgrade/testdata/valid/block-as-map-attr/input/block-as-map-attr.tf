resource "test_instance" "foo" {
  network = {
    cidr_block = "10.1.0.0/16"
  }
}
