variable "foo" {}

resource "test_instance" "foo" {
    foo = "${var.foo}"

    # This is here because at some point it caused a test failure
    network_interface {
      device_index = 0
      description = "Main network interface"
    }
}
