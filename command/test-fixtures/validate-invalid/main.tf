resorce "test_instance" "foo" { # Intentional typo to test error reporting
    ami = "bar"

    network_interface {
      device_index = 0
      description = "Main network interface"
    }
}
