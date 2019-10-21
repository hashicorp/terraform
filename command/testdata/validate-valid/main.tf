variable "var_with_escaped_interp" {
  # This is here because in the past it failed. See Github #13001
  default = "foo-$${bar.baz}"
}

resource "test_instance" "foo" {
    ami = "bar"

    # This is here because at some point it caused a test failure
    network_interface {
      device_index = 0
      description = "Main network interface"
    }
}
