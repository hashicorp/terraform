resource "test_instance" "foo" {
  count = 3
  ami   = "bar"

  # This is here because at some point it caused a test failure
  network_interface {
    device_index = 0
    description  = "Main network interface"
  }
}
