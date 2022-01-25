terraform {
  backend "http" {
  }
}

resource "test_instance" "foo" {
  ami = "bar"
}
