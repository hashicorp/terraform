resource "test_instance" "foo" {
  count = 1
  ami = "bar"
}

data "test_ds" "bar" {
  filter = "foo"
}
