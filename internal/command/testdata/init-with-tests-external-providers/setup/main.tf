terraform {
  required_providers {
    configure = {
      source = "testing/configure"
    }
  }
}

resource "configure_instance" "baz" {
  ami = "baz"
}
