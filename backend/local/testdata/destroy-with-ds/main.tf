resource "test_instance" "foo" {
  ami = "bar"
}

data "test_ds" "bar" {
  filter = "foo"
}
