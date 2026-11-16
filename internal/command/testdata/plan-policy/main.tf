resource "test_instance" "foo" {
  ami = "bar"

  network_interface {
    device_index = 0
    description  = "Main network interface"
  }
  depends_on = [data.test_data_source.a]
}

data "test_data_source" "a" {
  id = "zzzzz"
}
