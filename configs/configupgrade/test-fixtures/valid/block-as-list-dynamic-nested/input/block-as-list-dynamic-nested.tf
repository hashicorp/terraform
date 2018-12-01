resource "test_instance" "foo" {
  network {
    subnet = "${var.baz}"
  }
}
