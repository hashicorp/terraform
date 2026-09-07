terraform {
  backend "inmem" {
  }
}

resource "test_instance" "foo" {
  ami = "bar"
}
